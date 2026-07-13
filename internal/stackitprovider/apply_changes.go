package stackitprovider

import (
	"context"
	"fmt"
	"sync"

	stackitdnsclient "github.com/stackitcloud/stackit-sdk-go/services/dns/v1api"
	"go.uber.org/zap"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
)

// ApplyChanges applies a given set of DNS changes to the STACKIT DNS API.
// It enforces a strict phase-based execution order to prevent orphaned records
// and mitigate quota limit issues (e.g., max 10k records per zone).
// Deletions are processed before creations to free up zone quota.
func (d *StackitDNSProvider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	if len(changes.Create)+len(changes.UpdateNew)+len(changes.Delete) == 0 {
		return nil
	}

	d.logger.Info("records to delete", zap.String("records", fmt.Sprintf("%v", changes.Delete)))

	zones, err := d.zoneFetcherClient.zones(ctx)
	if err != nil {
		return err
	}

	// Separate ownership records (TXT) from target records (A, CNAME, etc.)
	// to enforce strict dependency ordering and prevent orphaned records.
	deleteTXT, deleteOther := splitTXTAndOther(changes.Delete)
	updateTXT, updateOther := splitTXTAndOther(changes.UpdateNew)
	createTXT, createOther := splitTXTAndOther(changes.Create)

	// Execution order is critical.
	// 1. Delete targets first, then their TXT ownership records.
	// 2. Update TXT ownerships, then targets.
	// 3. Create TXT ownerships first, then create targets.
	batches := [][]changeTask{
		d.buildRRSetTasks(deleteOther, DELETE),
		d.buildRRSetTasks(deleteTXT, DELETE),
		d.buildRRSetTasks(updateTXT, UPDATE),
		d.buildRRSetTasks(updateOther, UPDATE),
		d.buildRRSetTasks(createTXT, CREATE),
		d.buildRRSetTasks(createOther, CREATE),
	}

	for _, batch := range batches {
		if len(batch) == 0 {
			continue
		}

		// If any batch fails (e.g., hitting a quota limit), the entire sync loop aborts.
		// This leaves the DNS state consistent for the next retry attempt.
		if err := d.handleRRSetWithWorkers(ctx, batch, zones); err != nil {
			return err
		}
	}

	return nil
}

// splitTXTAndOther separates TXT records from all other record types.
// External-DNS relies on TXT records to track ownership.
func splitTXTAndOther(endpoints []*endpoint.Endpoint) ([]*endpoint.Endpoint, []*endpoint.Endpoint) {
	var txt, other []*endpoint.Endpoint
	for _, ep := range endpoints {
		if ep.RecordType == "TXT" {
			txt = append(txt, ep)
		} else {
			other = append(other, ep)
		}
	}
	return txt, other
}

// buildRRSetTasks wraps endpoint changes into executable tasks for the worker pool.
func (d *StackitDNSProvider) buildRRSetTasks(
	endpoints []*endpoint.Endpoint,
	action string,
) []changeTask {
	tasks := make([]changeTask, 0, len(endpoints))

	for _, change := range endpoints {
		tasks = append(tasks, changeTask{
			action: action,
			change: change,
		})
	}

	return tasks
}

// handleRRSetWithWorkers processes a batch of DNS changes concurrently.
// It implements a fail-fast mechanism: if any worker encounters an error
// (like a 4xx quota limit reached), it cancels the context to stop remaining queued tasks,
// preventing an API DoS.
func (d *StackitDNSProvider) handleRRSetWithWorkers(
	ctx context.Context,
	tasks []changeTask,
	zones []stackitdnsclient.Zone,
) error {
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	workerChannel := make(chan changeTask, len(tasks))
	errorChannel := make(chan error, len(tasks))

	var wg sync.WaitGroup
	for i := 0; i < d.workers; i++ {
		wg.Add(1)
		go d.changeWorker(cancelCtx, workerChannel, errorChannel, zones, &wg)
	}

	for _, task := range tasks {
		workerChannel <- task
	}
	close(workerChannel)

	var firstErr error
	for i := 0; i < len(tasks); i++ {
		err := <-errorChannel
		if err != nil && firstErr == nil {
			if err != context.Canceled {
				firstErr = err
				d.logger.Error("error encountered during batch processing, cancelling remaining tasks", zap.Error(err))
				// Fail fast: signal all active and pending workers to abort.
				cancel()
			}
		}
	}

	// wait until all workers have finished
	wg.Wait()

	return firstErr
}

// changeWorker listens for tasks on the workerChannel and executes the appropriate API call.
// It respects context cancellation to safely abort pending operations.
func (d *StackitDNSProvider) changeWorker(
	ctx context.Context,
	changes <-chan changeTask,
	errorChannel chan<- error,
	zones []stackitdnsclient.Zone,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for change := range changes {
		// Check for context cancellation before processing the next task.
		if err := ctx.Err(); err != nil {
			errorChannel <- err
			continue
		}

		var err error
		switch change.action {
		case CREATE:
			err = d.createRRSet(ctx, change.change, zones)
		case UPDATE:
			err = d.updateRRSet(ctx, change.change, zones)
		case DELETE:
			err = d.deleteRRSet(ctx, change.change, zones)
		}
		errorChannel <- err
	}

	d.logger.Debug("change worker finished")
}

// createRRSet creates a new record set in the stackitprovider for the given endpoint.
func (d *StackitDNSProvider) createRRSet(
	ctx context.Context,
	change *endpoint.Endpoint,
	zones []stackitdnsclient.Zone,
) error {
	resultZone, found := findBestMatchingZone(change.DNSName, zones)
	if !found {
		return fmt.Errorf("no matching zone found for %s", change.DNSName)
	}

	logFields := getLogFields(change, CREATE, resultZone.Id)
	d.logger.Info("create record set", logFields...)

	if d.dryRun {
		d.logger.Debug("dry run, skipping", logFields...)

		return nil
	}

	modifyChange(change)

	rrSetPayload := getStackitRecordSetPayload(change)

	// ignore all errors to just retry on next run
	_, err := d.apiClient.DefaultAPI.CreateRecordSet(ctx, d.projectId, resultZone.Id).CreateRecordSetPayload(rrSetPayload).Execute()
	if err != nil {
		d.logger.Error("error creating record set", zap.Error(err))

		return err
	}

	d.logger.Info("create record set successfully", logFields...)

	return nil
}

// updateRRSet patches (overrides) contents in the record set in the stackitprovider.
func (d *StackitDNSProvider) updateRRSet(
	ctx context.Context,
	change *endpoint.Endpoint,
	zones []stackitdnsclient.Zone,
) error {
	modifyChange(change)

	resultZone, resultRRSet, err := d.rrSetFetcherClient.getRRSetForUpdateDeletion(ctx, change, zones)
	if err != nil {
		return err
	}

	logFields := getLogFields(change, UPDATE, resultRRSet.Id)
	d.logger.Info("update record set", logFields...)

	if d.dryRun {
		d.logger.Debug("dry run, skipping", logFields...)

		return nil
	}

	rrSet := getStackitPartialUpdateRecordSetPayload(change)

	_, err = d.apiClient.DefaultAPI.PartialUpdateRecordSet(ctx, d.projectId, resultZone.Id, resultRRSet.Id).PartialUpdateRecordSetPayload(rrSet).Execute()
	if err != nil {
		d.logger.Error("error updating record set", zap.Error(err))

		return err
	}

	d.logger.Info("update record set successfully", logFields...)

	return nil
}

// deleteRRSet deletes a record set in the stackitprovider for the given endpoint.
func (d *StackitDNSProvider) deleteRRSet(
	ctx context.Context,
	change *endpoint.Endpoint,
	zones []stackitdnsclient.Zone,
) error {
	modifyChange(change)

	resultZone, resultRRSet, err := d.rrSetFetcherClient.getRRSetForUpdateDeletion(ctx, change, zones)
	if err != nil {
		return err
	}

	logFields := getLogFields(change, DELETE, resultRRSet.Id)
	d.logger.Info("delete record set", logFields...)

	if d.dryRun {
		d.logger.Debug("dry run, skipping", logFields...)

		return nil
	}

	_, err = d.apiClient.DefaultAPI.DeleteRecordSet(ctx, d.projectId, resultZone.Id, resultRRSet.Id).Execute()
	if err != nil {
		d.logger.Error("error deleting record set", zap.Error(err))

		return err
	}

	d.logger.Info("delete record set successfully", logFields...)

	return nil
}
