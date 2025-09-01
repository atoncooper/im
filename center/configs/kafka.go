package configs

import "crypto/tls"

type KafkaConfig struct {
	Broker []string
	Topic  string
}

type TLSConfig struct {
	Enable   bool
	CertFile string
	KeyFile  string
	CAFile   string
	Config   *tls.Config
}

func newDefaultKafka() *KafkaConfig {
	return &KafkaConfig{}
}

type KafkaParams struct {
	defaultParams *KafkaConfig
	defineParams  string
}

type KafkaParamsOpts func(*KafkaParams)

func DefaultKafkaParams(c *KafkaConfig) KafkaParamsOpts {
	return func(params *KafkaParams) {
		params.defaultParams = c
	}
}

func DefineKafkaParams(c string) KafkaParamsOpts {
	return func(kp *KafkaParams) {
		kp.defineParams = c
	}
}

func NewKafka(opts ...KafkaParamsOpts) {
	p := &KafkaParams{}
	for _, opt := range opts {
		opt(p)
	}

	if p.defaultParams == nil {
		p.defaultParams = newDefaultKafka()
	}

}
