package stackitprovider

import (
	"context"
	"fmt"

	stackitdnsclient "github.com/stackitcloud/stackit-sdk-go/services/dns"
	"go.uber.org/zap"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
)

// ApplyChanges applies a given set of changes in a given zone.
func (d *StackitDNSProvider) ApplyChanges(ctx context.Context, changes *plan.Changes) error {
	zones, err := d.zoneFetcherClient.zones(ctx)
	if err != nil {
		return err
	}

	// create rr set. POST /v1/projects/{projectId}/zones/{zoneId}/rrsets
	err = d.createRRSets(ctx, zones, changes.Create)
	if err != nil {
		return err
	}

	// update rr set. PATCH /v1/projects/{projectId}/zones/{zoneId}/rrsets/{rrSetId}
	err = d.updateRRSets(ctx, zones, changes.UpdateNew)
	if err != nil {
		return err
	}

	// delete rr set. DELETE /v1/projects/{projectId}/zones/{zoneId}/rrsets/{rrSetId}
	err = d.deleteRRSets(ctx, zones, changes.Delete)
	if err != nil {
		return err
	}

	return nil
}

// createRRSets creates new record sets in the stackitprovider for the given endpoints that are in the
// creation field.
func (d *StackitDNSProvider) createRRSets(
	ctx context.Context,
	zones []stackitdnsclient.Zone,
	endpoints []*endpoint.Endpoint,
) error {
	if len(endpoints) == 0 {
		return nil
	}

	return d.handleRRSetWithWorkers(ctx, endpoints, zones, CREATE)
}

// updateRRSets patches (overrides) contents in the record sets in the stackitprovider for the given
// endpoints that are in the update new field.
func (d *StackitDNSProvider) updateRRSets(
	ctx context.Context,
	zones []stackitdnsclient.Zone,
	endpoints []*endpoint.Endpoint,
) error {
	if len(endpoints) == 0 {
		return nil
	}

	return d.handleRRSetWithWorkers(ctx, endpoints, zones, UPDATE)
}

// deleteRRSets deletes record sets in the stackitprovider for the given endpoints that are in the
// deletion field.
func (d *StackitDNSProvider) deleteRRSets(
	ctx context.Context,
	zones []stackitdnsclient.Zone,
	endpoints []*endpoint.Endpoint,
) error {
	if len(endpoints) == 0 {
		d.logger.Debug("no endpoints to delete")

		return nil
	}

	d.logger.Info("records to delete", zap.String("records", fmt.Sprintf("%v", endpoints)))

	return d.handleRRSetWithWorkers(ctx, endpoints, zones, DELETE)
}

// handleRRSetWithWorkers handles the given endpoints with workers to optimize speed.
func (d *StackitDNSProvider) handleRRSetWithWorkers(
	ctx context.Context,
	endpoints []*endpoint.Endpoint,
	zones []stackitdnsclient.Zone,
	action string,
) error {
	workerChannel := make(chan changeTask, len(endpoints))
	errorChannel := make(chan error, len(endpoints))

	for i := 0; i < d.workers; i++ {
		go d.changeWorker(ctx, workerChannel, errorChannel, zones)
	}

	for _, change := range endpoints {
		workerChannel <- changeTask{
			action: action,
			change: change,
		}
	}

	for i := 0; i < len(endpoints); i++ {
		err := <-errorChannel
		if err != nil {
			close(workerChannel)

			return err
		}
	}

	close(workerChannel)

	return nil
}

// changeWorker is a worker that handles changes passed by a channel.
func (d *StackitDNSProvider) changeWorker(
	ctx context.Context,
	changes chan changeTask,
	errorChannel chan error,
	zones []stackitdnsclient.Zone,
) {
	for change := range changes {
		switch change.action {
		case CREATE:
			err := d.createRRSet(ctx, change.change, zones)
			errorChannel <- err
		case UPDATE:
			err := d.updateRRSet(ctx, change.change, zones)
			errorChannel <- err
		case DELETE:
			err := d.deleteRRSet(ctx, change.change, zones)
			errorChannel <- err
		}
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
