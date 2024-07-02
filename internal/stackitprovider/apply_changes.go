package stackitprovider

import (
	"context"
	"fmt"
	"sync"

	stackitdnsclient "github.com/stackitcloud/stackit-sdk-go/services/dns"
	"go.uber.org/zap"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
)

// ApplyChanges applies a given set of changes in a given zone.
func (d *StackitDNSProvider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	var tasks []changeTask
	// create rr set. POST /v1/projects/{projectId}/zones/{zoneId}/rrsets
	tasks = append(tasks, d.buildRRSetTasks(changes.Create, CREATE)...)
	// update rr set. PATCH /v1/projects/{projectId}/zones/{zoneId}/rrsets/{rrSetId}
	tasks = append(tasks, d.buildRRSetTasks(changes.UpdateNew, UPDATE)...)
	d.logger.Info("records to delete", zap.String("records", fmt.Sprintf("%v", changes.Delete)))
	// delete rr set. DELETE /v1/projects/{projectId}/zones/{zoneId}/rrsets/{rrSetId}
	tasks = append(tasks, d.buildRRSetTasks(changes.Delete, DELETE)...)

	zones, err := d.zoneFetcherClient.zones(ctx)
	if err != nil {
		return err
	}

	return d.handleRRSetWithWorkers(ctx, tasks, zones)
}

// handleRRSetWithWorkers handles the given endpoints with workers to optimize speed.
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

// handleRRSetWithWorkers handles the given endpoints with workers to optimize speed.
func (d *StackitDNSProvider) handleRRSetWithWorkers(
	ctx context.Context,
	tasks []changeTask,
	zones []stackitdnsclient.Zone,
) error {
	workerChannel := make(chan changeTask, len(tasks))
	errorChannel := make(chan error, len(tasks))

	var wg sync.WaitGroup
	for i := 0; i < d.workers; i++ {
		wg.Add(1)
		go d.changeWorker(ctx, workerChannel, errorChannel, zones, &wg)
	}

	for _, task := range tasks {
		workerChannel <- task
	}
	close(workerChannel)

	// capture first error
	var err error
	for i := 0; i < len(tasks); i++ {
		err = <-errorChannel
		if err != nil {
			break
		}
	}

	// wait until all workers have finished
	wg.Wait()

	return err
}

// changeWorker is a worker that handles changes passed by a channel.
func (d *StackitDNSProvider) changeWorker(
	ctx context.Context,
	changes chan changeTask,
	errorChannel chan error,
	zones []stackitdnsclient.Zone,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for change := range changes {
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

	logFields := getLogFields(change, CREATE, *resultZone.Id)
	d.logger.Info("create record set", logFields...)

	if d.dryRun {
		d.logger.Debug("dry run, skipping", logFields...)

		return nil
	}

	modifyChange(change)

	rrSetPayload := getStackitRecordSetPayload(change)

	// ignore all errors to just retry on next run
	_, err := d.apiClient.CreateRecordSet(ctx, d.projectId, *resultZone.Id).CreateRecordSetPayload(rrSetPayload).Execute()
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

	logFields := getLogFields(change, UPDATE, *resultRRSet.Id)
	d.logger.Info("update record set", logFields...)

	if d.dryRun {
		d.logger.Debug("dry run, skipping", logFields...)

		return nil
	}

	rrSet := getStackitPartialUpdateRecordSetPayload(change)

	_, err = d.apiClient.PartialUpdateRecordSet(ctx, d.projectId, *resultZone.Id, *resultRRSet.Id).PartialUpdateRecordSetPayload(rrSet).Execute()
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

	logFields := getLogFields(change, DELETE, *resultRRSet.Id)
	d.logger.Info("delete record set", logFields...)

	if d.dryRun {
		d.logger.Debug("dry run, skipping", logFields...)

		return nil
	}

	_, err = d.apiClient.DeleteRecordSet(ctx, d.projectId, *resultZone.Id, *resultRRSet.Id).Execute()
	if err != nil {
		d.logger.Error("error deleting record set", zap.Error(err))

		return err
	}

	d.logger.Info("delete record set successfully", logFields...)

	return nil
}
