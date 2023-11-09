package server

import (
	gometrics "github.com/armon/go-metrics"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/telemetry"
	bridgeapi "github.com/dydxprotocol/v4-chain/protocol/daemons/bridge/api"
	"github.com/dydxprotocol/v4-chain/protocol/daemons/constants"
	liquidationapi "github.com/dydxprotocol/v4-chain/protocol/daemons/liquidation/api"
	pricefeedapi "github.com/dydxprotocol/v4-chain/protocol/daemons/pricefeed/api"
	"github.com/dydxprotocol/v4-chain/protocol/daemons/server/types"
	daemontypes "github.com/dydxprotocol/v4-chain/protocol/daemons/types"
	"github.com/dydxprotocol/v4-chain/protocol/lib/metrics"
	"net"
	"syscall"
	"time"
)

// Server struct defines the shared gRPC server for all daemons.
// The struct contains fields related to gRPC server that are common to all daemons.
// In addition, the struct contains fields that are specific to various daemon services.
// needed for various services.
type Server struct {
	logger        log.Logger
	gsrv          daemontypes.GrpcServer
	fileHandler   daemontypes.FileHandler
	socketAddress string

	updateMonitor *types.HealthMonitor

	BridgeServer
	PriceFeedServer
	LiquidationServer
}

// NewServer creates a single gRPC server that's shared across multiple daemons for communication.
// uniqueTestIdentifier is a string that can be used to identify services spawned by a particular test case,
// so that they can be cleaned up after the test case is complete.
func NewServer(
	logger log.Logger,
	grpcServer daemontypes.GrpcServer,
	fileHandler daemontypes.FileHandler,
	socketAddress string,
) *Server {
	return &Server{
		logger:        logger,
		gsrv:          grpcServer,
		fileHandler:   fileHandler,
		socketAddress: socketAddress,
		updateMonitor: types.NewHealthMonitor(types.DaemonStartupGracePeriod, logger),
	}
}

// Stop stops the daemon server's gRPC service.
func (server *Server) Stop() {
	server.updateMonitor.Stop()
	server.gsrv.Stop()
}

// DisableUpdateMonitoringForTesting disables the update monitor for testing purposes. This is needed in integration
// tests that do not run the full protocol.
func (server *Server) DisableUpdateMonitoringForTesting() {
	server.updateMonitor.DisableForTesting()
}

// registerDaemon registers a daemon service with the update monitor.
func (server *Server) registerDaemon(
	daemonKey string,
	maximumAcceptableUpdateDelay time.Duration,
) {
	if err := server.updateMonitor.RegisterDaemonService(daemonKey, maximumAcceptableUpdateDelay); err != nil {
		server.logger.Error(
			"Failed to register daemon service with update monitor",
			"error",
			err,
			"service",
			daemonKey,
			"maximumAcceptableUpdateDelay",
			maximumAcceptableUpdateDelay,
		)
		panic(err)
	}
}

// reportResponse reports a response from a daemon service with the update monitor. This is used to
// ensure that the daemon continues to operate. If the update monitor does not see a response from a
// registered daemon within the maximumAcceptableUpdateDelay, it will cause the protocol to panic.
func (server *Server) reportResponse(
	daemonKey string,
) error {
	telemetry.IncrCounterWithLabels(
		[]string{
			metrics.DaemonServer,
			metrics.ValidResponse,
		},
		1,
		[]gometrics.Label{
			metrics.GetLabelForStringValue(metrics.Daemon, daemonKey),
		},
	)
	return server.updateMonitor.RegisterValidResponse(daemonKey)
}

// Start clears the current socket and establishes a new socket connection
// on the local filesystem.
// See URL for more information: https://eli.thegreenplace.net/2019/unix-domain-sockets-in-go/
func (server *Server) Start() {
	if err := server.fileHandler.RemoveAll(server.socketAddress); err != nil {
		server.logger.Error("Failed to clear socket for daemon gRPC server", "error", err)
		panic(err)
	}

	// Restrict so that only user can read or write to socket generated by `net.Listen`.
	oldValue := syscall.Umask(constants.UmaskUserReadWriteOnly)

	ln, err := net.Listen(constants.UnixProtocol, server.socketAddress)

	// Restore umask bits back to previous value so that the entire process is not restricted to `UmaskUserReadWriteOnly`.
	syscall.Umask(oldValue)

	if err != nil {
		server.logger.Error("Failed to listen to daemon gRPC server", "error", err)
		panic(err)
	}

	server.logger.Info("Daemon gRPC server is listening", "address", ln.Addr())

	// Register gRPC services needed by the daemons. This is required before invoking `Serve`.
	// https://pkg.go.dev/google.golang.org/grpc#Server.RegisterService

	// Register Server to ingest gRPC requests from bridge daemon.
	bridgeapi.RegisterBridgeServiceServer(server.gsrv, server)

	// Register Server to ingest gRPC requests from price feed daemon and update market prices.
	pricefeedapi.RegisterPriceFeedServiceServer(server.gsrv, server)

	// Register Server to ingest gRPC requests from liquidation daemon.
	liquidationapi.RegisterLiquidationServiceServer(server.gsrv, server)

	if err := server.gsrv.Serve(ln); err != nil {
		server.logger.Error("daemon gRPC server stopped with an error", "error", err)
		panic(err)
	}
}
