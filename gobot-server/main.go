package main // import "github.com/robbert229/filament-reloaded-server/gobot-server"

import (
	"context"
	"flag"
	"fmt"
	"github.com/robbert229/filament-reloaded-server/gobot-server/pb"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"gobot.io/x/gobot"
	"gobot.io/x/gobot/drivers/gpio"
	"gobot.io/x/gobot/platforms/firmata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
)

var (
	pinsString = flag.String("pins", "2", "the comma separated list of pins to use as filament sensors")
	addrString = flag.String("addr", "127.0.0.1:3875", "The address to listen on")
	deviceString = flag.String("dev", "/dev/ttyACM0", "The device to communicate with")

	logLevelString = flag.String("log.level", "DEBUG", "The level of messages to log PANIC|ERROR|INFO|DEBUG|WARN")
	logFmtString = flag.String("log.format", "TEXT", "The log formatter to use TEXT|JSON")
)

func getLogger() *logrus.Logger {
	logger := logrus.New()

	switch *logLevelString {
	case "INFO":
		logger.Level = logrus.InfoLevel
	case "PANIC":
		logger.Level = logrus.PanicLevel
	case "DEBUG":
		logger.Level = logrus.DebugLevel
	case "ERROR":
		logger.Level = logrus.ErrorLevel
	case "WARN":
		logger.Level = logrus.WarnLevel
	default:
		fmt.Fprintf(os.Stderr, "invalid log level given: '%s'\n", *logLevelString)

		flag.Usage()
		os.Exit(-1)
	}

	switch *logFmtString {
	case "TEXT":
		logger.Formatter = &logrus.TextFormatter{}
	case "JSON":
		logger.Formatter = &logrus.JSONFormatter{}
	default:
		fmt.Fprintf(os.Stderr, "invalid log formatter given: '%s'\n", *logFmtString)
	}

	return logger
}

type pbServer struct {
	offChan <- chan string
	onChan <- chan string
	buttons map[string]*gpio.ButtonDriver
}

// Listen is the protobuf way of listening for events.
func (s pbServer) Listen(req *pb.ListenRequest, server pb.Listen_ListenServer) error {
	return nil
}


// Check is the protobuf way of checking an endpoint.
func (s pbServer) Check(ctx context.Context, req *pb.CheckRequest) (*pb.CheckResponse, error) {
	button, ok := s.buttons[fmt.Sprintf("%d", req.Pin)]
	if !ok {
		return nil, status.Error(codes.NotFound, "pin not installed")
	}

	pinState := pb.PinState_INVALID
	if button.Active {
		pinState = pb.PinState_HIGH
	} else {
		pinState = pb.PinState_LOW
	}

	return &pb.CheckResponse{
		State: pinState,
	}, nil
}


// Server is a robot that is used to manage
type Server struct {
	OffChan chan string
	OnChan chan string
	Buttons map[string]*gpio.ButtonDriver

	Robot  *gobot.Robot
	Addr string
	Logger *logrus.Logger
}

// NewServer returns a new server that exposes the data over grpc and http.
func NewServer(logger *logrus.Logger, addr, pinsString, device string) *Server {
	pins  := map[string]*gpio.ButtonDriver{}
	devices := []gobot.Device{}

	pinsStringSlice := strings.Split(pinsString, ",")
	if len(pinsStringSlice) == 0 {
		logger.Error("no pins given")
		os.Exit(1)
	}

	firmataAdaptor := firmata.NewAdaptor(device)

	for _, pinsStringToken := range pinsStringSlice {
		_, err := strconv.ParseInt(pinsStringToken, 10, 8)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid pins: '%s'\n", pinsStringToken)
			flag.Usage()
			os.Exit(1)
		}

		button := gpio.NewButtonDriver(firmataAdaptor, pinsStringToken)

		pins[pinsStringToken] = button
		devices = append(devices, button)
	}

	offChan := make(chan string)
	onChan := make(chan string)

	work := func() {
		for pin, button := range pins {
			button.On(gpio.ButtonPush, func(data interface{}) {
				logger.WithFields(logrus.Fields{"pin": pin, "state": "off"}).Debug("handling button event")
				onChan <- pin
			})

			button.On(gpio.ButtonRelease, func(data interface{}) {
				logger.WithFields(logrus.Fields{"pin": pin, "state": "on"}).Debug("handling button event")
				offChan <- pin
			})
		}
	}

	return &Server{
		OffChan: offChan,
		OnChan: onChan,
		Addr: addr,
		Logger: logger,
		Buttons: pins,
		Robot: gobot.NewRobot("sensor",
			[]gobot.Connection{firmataAdaptor},
			devices,
			work,
		),
	}
}

func (s Server) listenServer() pb.ListenServer {
	return &pbServer{
		offChan: s.OffChan,
		onChan: s.OnChan,
		buttons: s.Buttons,
	}
}

func (s Server) buildGRPCServer() *grpc.Server {
	server := grpc.NewServer()
	pb.RegisterListenServer(server, s.listenServer())
	return server
}


func (s Server) buildHttpServer() *http.Server{
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}

	err := pb.RegisterListenHandlerFromEndpoint(context.Background(), mux, s.Addr, opts)
	if err != nil {
		panic("should be able to connect to exposed address")
	}

	return &http.Server{
		Handler: mux,
	}
}

func (s Server) Serve() error {
	lis, err := net.Listen("tcp", *addrString)
	if err != nil {
		logrus.WithField("addr", s.Addr).Errorf("failed to listen: %+v", err)
	}

	logrus.WithField("addr", s.Addr).Info("listening on address")

	m := cmux.New(lis)
	grpcL := m.Match(cmux.HTTP2HeaderField("content-type", "application/grpc"))
	httpL := m.Match(cmux.Any())

	grpcS := s.buildGRPCServer()
	httpS := s.buildHttpServer()

	errChan := make(chan error)

	go func() {
		err := grpcS.Serve(grpcL)
		errChan <- errors.WithStack(err)
	}()

	go func() {
		err := httpS.Serve(httpL)
		errChan <- errors.WithStack(err)
	}()

	go func() {
		err := s.Robot.Start()
		errChan <- errors.WithStack(err)
	}()

	go func(){
		err := m.Serve()
		errChan <- errors.WithStack(err)
	}()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Interrupt)
		<- c
		errChan <- nil
	}()

	err = <- errChan

	waitChan := make(chan error)

	go func(){
		if err := s.Robot.Stop(); err != nil {
			s.Logger.Warn("error encountered when stopping gobot: %+v", err)
		}
		waitChan <- nil
	}()

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	select {
		case <- waitChan:
			return nil
		case <- ticker.C:
			s.Logger.Warn("timeout occurred waiting for gobot to shutdown")
			return nil
	}
}

func main() {
	flag.Parse()

	logger := getLogger()

	server := NewServer(logger, *addrString, *pinsString, *deviceString)

	err := server.Serve()
	if err != nil {
		logger.Errorf("unhandled error: %+v", err)
	}
	logger.Info("server has shutdown")
}
