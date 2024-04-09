package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/Shopify/sarama"
	kingpin "github.com/alecthomas/kingpin/v2"
	klog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grafana/kafka_exporter/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	plog "github.com/prometheus/common/promlog"
	plogflag "github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/rcrowley/go-metrics"
)

func init() {
	metrics.UseNilMetrics = true
	prometheus.MustRegister(version.NewCollector("kafka_exporter"))
}

// hack around flag.Parse and klog.init flags
func toFlagString(name string, help string, value string) *string {
	flag.CommandLine.String(name, value, help) // hack around flag.Parse and klog.init flags
	return kingpin.Flag(name, help).Default(value).String()
}

func toFlagBool(name string, help string, value bool, valueString string) *bool {
	flag.CommandLine.Bool(name, value, help) // hack around flag.Parse and klog.init flags
	return kingpin.Flag(name, help).Default(valueString).Bool()
}

func toFlagStringsVar(name string, help string, value string, target *[]string) {
	flag.CommandLine.String(name, value, help) // hack around flag.Parse and klog.init flags
	kingpin.Flag(name, help).Default(value).StringsVar(target)
}

func toFlagStringVar(name string, help string, value string, target *string) {
	flag.CommandLine.String(name, value, help) // hack around flag.Parse and klog.init flags
	kingpin.Flag(name, help).Default(value).StringVar(target)
}

func toFlagBoolVar(name string, help string, value bool, valueString string, target *bool) {
	flag.CommandLine.Bool(name, value, help) // hack around flag.Parse and klog.init flags
	kingpin.Flag(name, help).Default(valueString).BoolVar(target)
}

func toFlagIntVar(name string, help string, value int, valueString string, target *int) {
	flag.CommandLine.Int(name, value, help) // hack around flag.Parse and klog.init flags
	kingpin.Flag(name, help).Default(valueString).IntVar(target)
}

func main() {
	var (
		listenAddress = toFlagString("web.listen-address", "Address to listen on for web interface and telemetry.", ":9308")
		metricsPath   = toFlagString("web.telemetry-path", "Path under which to expose metrics.", "/metrics")
		topicFilter   = toFlagString("topic.filter", "Regex that determines which topics to collect.", ".*")
		topicExclude  = toFlagString("topic.exclude", "Regex that determines which topics to exclude.", "^$")
		groupFilter   = toFlagString("group.filter", "Regex that determines which consumer groups to collect.", ".*")
		groupExclude  = toFlagString("group.exclude", "Regex that determines which consumer groups to exclude.", "^$")
		logSarama     = toFlagBool("log.enable-sarama", "Turn on Sarama logging, default is false.", false, "false")

		opts = exporter.Options{}
	)

	toFlagStringsVar("kafka.server", "Address (host:port) of Kafka server.", "kafka:9092", &opts.Uri)
	toFlagBoolVar("sasl.enabled", "Connect using SASL/PLAIN, default is false.", false, "false", &opts.UseSASL)
	toFlagBoolVar("sasl.handshake", "Only set this to false if using a non-Kafka SASL proxy, default is true.", true, "true", &opts.UseSASLHandshake)
	toFlagStringVar("sasl.username", "SASL user name.", "", &opts.SaslUsername)
	toFlagStringVar("sasl.password", "SASL user password.", "", &opts.SaslPassword)
	toFlagStringVar("sasl.mechanism", "The SASL SCRAM SHA algorithm sha256 or sha512 or gssapi as mechanism", "", &opts.SaslMechanism)
	toFlagStringVar("sasl.service-name", "Service name when using kerberos Auth", "", &opts.ServiceName)
	toFlagStringVar("sasl.kerberos-config-path", "Kerberos config path", "", &opts.KerberosConfigPath)
	toFlagStringVar("sasl.realm", "Kerberos realm", "", &opts.Realm)
	toFlagStringVar("sasl.kerberos-auth-type", "Kerberos auth type. Either 'keytabAuth' or 'userAuth'", "", &opts.KerberosAuthType)
	toFlagStringVar("sasl.keytab-path", "Kerberos keytab file path", "", &opts.KeyTabPath)
	toFlagBoolVar("sasl.disable-PA-FX-FAST", "Configure the Kerberos client to not use PA_FX_FAST, default is false.", false, "false", &opts.SaslDisablePAFXFast)
	toFlagBoolVar("tls.enabled", "Connect to Kafka using TLS, default is false.", false, "false", &opts.UseTLS)
	toFlagStringVar("tls.server-name", "Used to verify the hostname on the returned certificates unless tls.insecure-skip-tls-verify is given. The kafka server's name should be given.", "", &opts.TlsServerName)
	toFlagStringVar("tls.ca-file", "The optional certificate authority file for Kafka TLS client authentication.", "", &opts.TlsCAFile)
	toFlagStringVar("tls.cert-file", "The optional certificate file for Kafka client authentication.", "", &opts.TlsCertFile)
	toFlagStringVar("tls.key-file", "The optional key file for Kafka client authentication.", "", &opts.TlsKeyFile)
	toFlagBoolVar("server.tls.enabled", "Enable TLS for web server, default is false.", false, "false", &opts.ServerUseTLS)
	toFlagBoolVar("server.tls.mutual-auth-enabled", "Enable TLS client mutual authentication, default is false.", false, "false", &opts.ServerMutualAuthEnabled)
	toFlagStringVar("server.tls.ca-file", "The certificate authority file for the web server.", "", &opts.ServerTlsCAFile)
	toFlagStringVar("server.tls.cert-file", "The certificate file for the web server.", "", &opts.ServerTlsCertFile)
	toFlagStringVar("server.tls.key-file", "The key file for the web server.", "", &opts.ServerTlsKeyFile)
	toFlagBoolVar("tls.insecure-skip-tls-verify", "If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure. Default is false", false, "false", &opts.TlsInsecureSkipTLSVerify)
	toFlagStringVar("kafka.version", "Kafka broker version", sarama.V2_0_0_0.String(), &opts.KafkaVersion)
	toFlagBoolVar("use.consumelag.zookeeper", "if you need to use a group from zookeeper, default is false", false, "false", &opts.UseZooKeeperLag)
	toFlagStringsVar("zookeeper.server", "Address (hosts) of zookeeper server.", "localhost:2181", &opts.UriZookeeper)
	toFlagStringVar("kafka.labels", "Kafka cluster name", "", &opts.Labels)
	toFlagStringVar("refresh.metadata", "Metadata refresh interval", "30s", &opts.MetadataRefreshInterval)
	toFlagBoolVar("offset.show-all", "Whether show the offset/lag for all consumer group, otherwise, only show connected consumer groups, default is true", true, "true", &opts.OffsetShowAll)
	toFlagBoolVar("concurrent.enable", "If true, all scrapes will trigger kafka operations otherwise, they will share results. WARN: This should be disabled on large clusters. Default is false", false, "false", &opts.AllowConcurrent)
	toFlagIntVar("topic.workers", "Number of topic workers", 100, "100", &opts.TopicWorkers)
	toFlagBoolVar("kafka.allow-auto-topic-creation", "If true, the broker may auto-create topics that we requested which do not already exist, default is false.", false, "false", &opts.AllowAutoTopicCreation)
	toFlagIntVar("max.offsets", "Maximum number of offsets to store in the interpolation table for a partition", 1000, "1000", &opts.MaxOffsets)

	plConfig := plog.Config{}
	plogflag.AddFlags(kingpin.CommandLine, &plConfig)
	kingpin.Version(version.Print("kafka_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	if *logSarama {
		sarama.Logger = log.New(os.Stdout, "[sarama] ", log.LstdFlags)
	}

	setup(*listenAddress, *metricsPath, *topicFilter, *topicExclude, *groupFilter, *groupExclude, *logSarama, opts)
}

func setup(
	listenAddress string,
	metricsPath string,
	topicFilter string,
	topicExclude string,
	groupFilter string,
	groupExclude string,
	logSarama bool,
	opts exporter.Options,
) {
	w := klog.NewSyncWriter(os.Stdout)
	logger := klog.NewLogfmtLogger(w)

	level.Info(logger).Log("Starting kafka_exporter", version.BuildContext())
	level.Debug(logger).Log("Build context", version.BuildContext())

	exp, err := exporter.New(logger, opts, topicFilter, topicExclude, groupFilter, groupExclude)
	if err != nil {
		level.Error(logger).Log(err.Error())
		return
	}
	defer exp.Close()
	prometheus.MustRegister(exp)

	http.Handle(metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`<html>
	        <head><title>Kafka Exporter</title></head>
	        <body>
	        <h1>Kafka Exporter</h1>
	        <p><a href='` + metricsPath + `'>Metrics</a></p>
	        </body>
	        </html>`))
		if err != nil {
			level.Error(logger).Log("Error handle / request", err)
		}
	})
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		// need more specific sarama check
		_, err := w.Write([]byte("ok"))
		if err != nil {
			level.Error(logger).Log("Error handle /healthz request", err)
		}
	})

	if opts.ServerUseTLS {
		level.Info(logger).Log("Listening on HTTPS", listenAddress)

		_, err := exporter.CanReadCertAndKey(opts.ServerTlsCertFile, opts.ServerTlsKeyFile)
		if err != nil {
			level.Error(logger).Log("error reading server cert and key")
		}

		clientAuthType := tls.NoClientCert
		if opts.ServerMutualAuthEnabled {
			clientAuthType = tls.RequireAndVerifyClientCert
		}

		certPool := x509.NewCertPool()
		if opts.ServerTlsCAFile != "" {
			if caCert, err := ioutil.ReadFile(opts.ServerTlsCAFile); err == nil {
				certPool.AppendCertsFromPEM(caCert)
			} else {
				level.Error(logger).Log("error reading server ca")
			}
		}

		tlsConfig := &tls.Config{
			ClientCAs:                certPool,
			ClientAuth:               clientAuthType,
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
			},
		}
		server := &http.Server{
			Addr:      listenAddress,
			TLSConfig: tlsConfig,
		}
		level.Error(logger).Log(server.ListenAndServeTLS(opts.ServerTlsCertFile, opts.ServerTlsKeyFile))
	} else {
		level.Info(logger).Log("Listening on HTTP", listenAddress)
		level.Error(logger).Log(http.ListenAndServe(listenAddress, nil))
	}
}
