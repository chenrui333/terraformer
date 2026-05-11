// SPDX-License-Identifier: Apache-2.0

//nolint:staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/qldb"
	qldbtypes "github.com/aws/aws-sdk-go-v2/service/qldb/types"
	"github.com/chenrui333/terraformer/terraformutils"
)

const (
	qldbLedgerResourceType = "aws_qldb_ledger"
	qldbStreamResourceType = "aws_qldb_stream"
)

var qldbAllowEmptyValues = []string{"tags."}

type QLDBGenerator struct {
	AWSService
}

func (g *QLDBGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}
	svc := qldb.NewFromConfig(config)
	return g.loadLedgers(svc)
}

func (g *QLDBGenerator) loadLedgers(svc *qldb.Client) error {
	p := qldb.NewListLedgersPaginator(svc, &qldb.ListLedgersInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return err
		}
		for _, ledger := range page.Ledgers {
			ledgerName := StringValue(ledger.Name)
			if resource, ok := newQLDBLedgerResource(ledgerName); ok {
				g.Resources = append(g.Resources, resource)
			}
			if ledgerName == "" {
				continue
			}
			if err := g.loadStreams(svc, ledgerName); err != nil {
				log.Printf("[WARN] Skipping QLDB streams for ledger %s: %v", ledgerName, err)
			}
		}
	}
	return nil
}

func (g *QLDBGenerator) loadStreams(svc *qldb.Client, ledgerName string) error {
	p := qldb.NewListJournalKinesisStreamsForLedgerPaginator(svc, &qldb.ListJournalKinesisStreamsForLedgerInput{
		LedgerName: &ledgerName,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(context.TODO())
		if qldbStreamNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, stream := range page.Streams {
			if resource, ok := newQLDBStreamResource(ledgerName, stream); ok {
				g.Resources = append(g.Resources, resource)
			}
		}
	}

	return nil
}

func newQLDBLedgerResource(ledgerName string) (terraformutils.Resource, bool) {
	if ledgerName == "" {
		return terraformutils.Resource{}, false
	}
	return terraformutils.NewSimpleResource(
		qldbLedgerImportID(ledgerName),
		ledgerName,
		qldbLedgerResourceType,
		"aws",
		qldbAllowEmptyValues), true
}

func newQLDBStreamResource(ledgerName string, stream qldbtypes.JournalKinesisStreamDescription) (terraformutils.Resource, bool) {
	streamID := StringValue(stream.StreamId)
	if ledgerName == "" || streamID == "" || !qldbStreamImportable(stream.Status) {
		return terraformutils.Resource{}, false
	}
	streamName := StringValue(stream.StreamName)
	if streamName == "" {
		streamName = streamID
	}
	return terraformutils.NewResource(
		qldbStreamImportID(streamID),
		qldbResourceName("stream", ledgerName, streamName, streamID),
		qldbStreamResourceType,
		"aws",
		map[string]string{
			"ledger_name": ledgerName,
			"stream_name": streamName,
		},
		qldbAllowEmptyValues,
		map[string]interface{}{}), true
}

func qldbStreamImportable(status qldbtypes.StreamStatus) bool {
	switch status {
	case qldbtypes.StreamStatusCompleted, qldbtypes.StreamStatusCanceled, qldbtypes.StreamStatusFailed:
		return false
	default:
		return true
	}
}

func qldbLedgerImportID(ledgerName string) string {
	return ledgerName
}

func qldbStreamImportID(streamID string) string {
	return streamID
}

func qldbResourceName(parts ...string) string {
	cleanParts := []string{}
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, fmt.Sprintf("%d/%s", len(part), part))
		}
	}
	if len(cleanParts) == 0 {
		return "qldb-resource"
	}
	return strings.Join(cleanParts, "/")
}

func qldbStreamNotFound(err error) bool {
	var notFound *qldbtypes.ResourceNotFoundException
	return errors.As(err, &notFound)
}
