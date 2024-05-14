package options

import (
	"time"

	"github.com/tomas-qstarrs/nano/log"

	"github.com/tomas-qstarrs/nano/codec"
	"github.com/tomas-qstarrs/nano/component"
	"github.com/tomas-qstarrs/nano/persist"
	"github.com/tomas-qstarrs/nano/pipeline"
)

// Options contains some configurations for current node
type Options struct {
	Pipeline       pipeline.Pipeline
	MasterPersist  persist.Persist
	IsMaster       bool
	AdvertiseAddr  string
	RetryInterval  time.Duration
	TCPAddr        string
	DebugAddr      string
	MemberAddr     string
	Components     *component.Components
	Label          string
	HttpAddr       string
	TSLCertificate string
	TSLKey         string
	Logger         log.Logger
	Codec          codec.Codec
	Etcd           bool
}

var Default = &Options{}
