// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelarrowreceiver // import "github.com/open-telemetry/otel-arrow/collector/receiver/otelarrowreceiver"

import (
	"context"

	"github.com/open-telemetry/otel-arrow/collector/receiver/otelarrowreceiver/internal/metadata"
	"github.com/open-telemetry/otel-arrow/collector/receiver/otelarrowreceiver/internal/sharedcomponent"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

const (
	defaultGRPCEndpoint = "0.0.0.0:4317"

	defaultMemoryLimitMiB = 128
)

// NewFactory creates a new OTLP receiver factory.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithTraces(createTraces, metadata.TracesStability),
		receiver.WithMetrics(createMetrics, metadata.MetricsStability),
		receiver.WithLogs(createLog, metadata.LogsStability))
}

// createDefaultConfig creates the default configuration for receiver.
func createDefaultConfig() component.Config {
	return &Config{
		Protocols: Protocols{
			GRPC: configgrpc.GRPCServerSettings{
				NetAddr: confignet.NetAddr{
					Endpoint:  defaultGRPCEndpoint,
					Transport: "tcp",
				},
				// We almost write 0 bytes, so no need to tune WriteBufferSize.
				ReadBufferSize: 512 * 1024,
			},
			Arrow: ArrowSettings{
				MemoryLimitMiB: defaultMemoryLimitMiB,
			},
		},
	}
}

// createTraces creates a trace receiver based on provided config.
func createTraces(
	_ context.Context,
	set receiver.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (receiver.Traces, error) {
	oCfg := cfg.(*Config)
	r, err := receivers.GetOrAdd(oCfg, func() (*otelArrowReceiver, error) {
		return newOTelArrowReceiver(oCfg, set)
	})
	if err != nil {
		return nil, err
	}

	if err = r.Unwrap().registerTraceConsumer(nextConsumer); err != nil {
		return nil, err
	}
	return r, nil
}

// createMetrics creates a metrics receiver based on provided config.
func createMetrics(
	_ context.Context,
	set receiver.CreateSettings,
	cfg component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {
	oCfg := cfg.(*Config)
	r, err := receivers.GetOrAdd(oCfg, func() (*otelArrowReceiver, error) {
		return newOTelArrowReceiver(oCfg, set)
	})
	if err != nil {
		return nil, err
	}

	if err = r.Unwrap().registerMetricsConsumer(consumer); err != nil {
		return nil, err
	}
	return r, nil
}

// createLog creates a log receiver based on provided config.
func createLog(
	_ context.Context,
	set receiver.CreateSettings,
	cfg component.Config,
	consumer consumer.Logs,
) (receiver.Logs, error) {
	oCfg := cfg.(*Config)
	r, err := receivers.GetOrAdd(oCfg, func() (*otelArrowReceiver, error) {
		return newOTelArrowReceiver(oCfg, set)
	})
	if err != nil {
		return nil, err
	}

	if err = r.Unwrap().registerLogsConsumer(consumer); err != nil {
		return nil, err
	}
	return r, nil
}

// This is the map of already created OTLP receivers for particular configurations.
// We maintain this map because the Factory is asked trace and metric receivers separately
// when it gets CreateTracesReceiver() and CreateMetricsReceiver() but they must not
// create separate objects, they must use one otelArrowReceiver object per configuration.
// When the receiver is shutdown it should be removed from this map so the same configuration
// can be recreated successfully.
var receivers = sharedcomponent.NewSharedComponents[*Config, *otelArrowReceiver]()
