package nano

import (
	"time"

	"github.com/tomas-qstarrs/nano/codec"
	"github.com/tomas-qstarrs/nano/component"
	"github.com/tomas-qstarrs/nano/env"
	"github.com/tomas-qstarrs/nano/log"
	"github.com/tomas-qstarrs/nano/message"
	"github.com/tomas-qstarrs/nano/options"
	"github.com/tomas-qstarrs/nano/persist"
	"github.com/tomas-qstarrs/nano/pipeline"
	"google.golang.org/grpc"
)

// Option defines a type for option, an option is a func operate options.NodeOptions
type Option func(*options.Options)

// WithPipeline sets the pipeline option.
func WithPipeline(pipeline pipeline.Pipeline) Option {
	return func(opt *options.Options) {
		opt.Pipeline = pipeline
	}
}

// WithAdvertiseAddr sets the advertise address option, it will be the listen address in
// master node and an advertise address which cluster member to connect
func WithAdvertiseAddr(addr string, retryInterval ...time.Duration) Option {
	return func(opt *options.Options) {
		opt.AdvertiseAddr = addr
		if len(retryInterval) > 0 {
			opt.RetryInterval = retryInterval[0]
		}
	}
}

// WithTCPAddr sets the listen address which is used to establish connection between
// cluster members. Will select an available port automatically if no member address
// setting and panic if no available port
func WithTCPAddr(addr string) Option {
	return func(opt *options.Options) {
		opt.TCPAddr = addr
	}
}

// WithDebugAddr works with debug http addr
func WithDebugAddr(addr string) Option {
	return func(opt *options.Options) {
		opt.DebugAddr = addr
	}
}

func WithMemberAddr(addr string) Option {
	return func(opt *options.Options) {
		opt.MemberAddr = addr
	}
}

// WithMaster sets the option to indicate whether the current node is master node
func WithMaster() Option {
	return func(opt *options.Options) {
		opt.IsMaster = true
	}
}

// WithGrpcOptions sets the grpc dial options
func WithGrpcOptions(opts ...grpc.DialOption) Option {
	return func(_ *options.Options) {
		env.GrpcOptions = append(env.GrpcOptions, opts...)
	}
}

// WithComponents sets the Components
func WithComponents(components *component.Components) Option {
	return func(opt *options.Options) {
		opt.Components = components
	}
}

// WithDebugMode makes 'nano' run under Debug mode.
func WithDebugMode(debug bool) Option {
	return func(_ *options.Options) {
		env.Debug = debug
	}
}

// WithSafeMode makes 'nano' run under Safe mode.
func WithSafeMode(safe bool) Option {
	return func(_ *options.Options) {
		env.Safe = safe
	}
}

// WithTimerPrecision sets the ticker precision, and time precision can not less
// than a Millisecond, and can not change after application running. The default
// precision is time.Second
func WithTimerPrecision(precision time.Duration) Option {
	if precision < time.Millisecond {
		panic("time precision can not less than a Millisecond")
	}
	return func(_ *options.Options) {
		env.TimerPrecision = precision
	}
}

// WithLabel sets the current node label in cluster
func WithLabel(label string) Option {
	return func(opt *options.Options) {
		opt.Label = label
	}
}

// WithVersion sets the current node version in cluster
func WithVersion(version string) Option {
	return func(opt *options.Options) {
		env.Version = version
		env.ShortVersion = message.ShortVersion(version)
	}
}

// WithHttpAddr sets the independent http address
func WithHttpAddr(httpAddr string) Option {
	return func(opt *options.Options) {
		opt.HttpAddr = httpAddr
	}
}

// WithMasterPersist sets the persist of cluster
func WithMasterPersist(persist persist.Persist) Option {
	return func(opt *options.Options) {
		opt.MasterPersist = persist
	}
}

// WithTSLConfig sets the `key` and `certificate` of TSL
func WithTSLConfig(certificate, key string) Option {
	return func(opt *options.Options) {
		opt.TSLCertificate = certificate
		opt.TSLKey = key
	}
}

// WithLogger overrides the default logger
func WithLogger(l log.Logger) Option {
	return func(opt *options.Options) {
		opt.Logger = l
	}
}

func WithCodec(c codec.Codec) Option {
	return func(opt *options.Options) {
		opt.Codec = c
	}
}

func WithEtcd() Option {
	return func(opt *options.Options) {
		opt.Etcd = true
	}
}

func WithWebSocketCompression() Option {
	return func(opt *options.Options) {
		env.WebSocketCompression = true
	}
}
