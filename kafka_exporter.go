package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	plog "github.com/prometheus/common/promlog"
	plogflag "github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/rcrowley/go-metrics"
	"gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/klog/v2"
)

const (
	namespace = "kafka"
	clientID  = "kafka_exporter"
)

func init() {
	metrics.UseNilMetrics = true
	prometheus.MustRegister(version.NewCollector("kafka_exporter"))
}

//func toFlag(name string, help string) *kingpin.FlagClause {
//	flag.CommandLine.String(name, "", help) // hack around flag.Parse and klog.init flags
//	return kingpin.Flag(name, help)
//}

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
		groupFilter   = toFlagString("group.filter", "Regex that determines which consumer groups to collect.", ".*")
		logSarama     = toFlagBool("log.enable-sarama", "Turn on Sarama logging, default is false.", false, "false")

		opts = KafkaOpts{}
	)

	toFlagStringsVar("kafka.server", "Address (host:port) of Kafka server.", "kafka:9092", &opts.uri)
	toFlagBoolVar("sasl.enabled", "Connect using SASL/PLAIN, default is false.", false, "false", &opts.useSASL)
	toFlagBoolVar("sasl.handshake", "Only set this to false if using a non-Kafka SASL proxy, default is true.", true, "true", &opts.useSASLHandshake)
	toFlagStringVar("sasl.username", "SASL user name.", "", &opts.saslUsername)
	toFlagStringVar("sasl.password", "SASL user password.", "", &opts.saslPassword)
	toFlagStringVar("sasl.mechanism", "The SASL SCRAM SHA algorithm sha256 or sha512 or gssapi as mechanism", "", &opts.saslMechanism)
	toFlagStringVar("sasl.service-name", "Service name when using kerberos Auth", "", &opts.serviceName)
	toFlagStringVar("sasl.kerberos-config-path", "Kerberos config path", "", &opts.kerberosConfigPath)
	toFlagStringVar("sasl.realm", "Kerberos realm", "", &opts.realm)
	toFlagStringVar("sasl.kerberos-auth-type", "Kerberos auth type. Either 'keytabAuth' or 'userAuth'", "", &opts.kerberosAuthType)
	toFlagStringVar("sasl.keytab-path", "Kerberos keytab file path", "", &opts.keyTabPath)
	toFlagBoolVar("sasl.disable-PA-FX-FAST", "Configure the Kerberos client to not use PA_FX_FAST, default is false.", false, "false", &opts.saslDisablePAFXFast)
	toFlagBoolVar("tls.enabled", "Connect to Kafka using TLS, default is false.", false, "false", &opts.useTLS)
	toFlagStringVar("tls.server-name", "Used to verify the hostname on the returned certificates unless tls.insecure-skip-tls-verify is given. The kafka server's name should be given.", "", &opts.tlsServerName)
	toFlagStringVar("tls.ca-file", "The optional certificate authority file for Kafka TLS client authentication.", "", &opts.tlsCAFile)
	toFlagStringVar("tls.cert-file", "The optional certificate file for Kafka client authentication.", "", &opts.tlsCertFile)
	toFlagStringVar("tls.key-file", "The optional key file for Kafka client authentication.", "", &opts.tlsKeyFile)
	toFlagBoolVar("server.tls.enabled", "Enable TLS for web server, default is false.", false, "false", &opts.serverUseTLS)
	toFlagBoolVar("server.tls.mutual-auth-enabled", "Enable TLS client mutual authentication, default is false.", false, "false", &opts.serverMutualAuthEnabled)
	toFlagStringVar("server.tls.ca-file", "The certificate authority file for the web server.", "", &opts.serverTlsCAFile)
	toFlagStringVar("server.tls.cert-file", "The certificate file for the web server.", "", &opts.serverTlsCertFile)
	toFlagStringVar("server.tls.key-file", "The key file for the web server.", "", &opts.serverTlsKeyFile)
	toFlagBoolVar("tls.insecure-skip-tls-verify", "If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure. Default is false", false, "false", &opts.tlsInsecureSkipTLSVerify)
	toFlagStringVar("kafka.version", "Kafka broker version", sarama.V2_0_0_0.String(), &opts.kafkaVersion)
	toFlagBoolVar("use.consumelag.zookeeper", "if you need to use a group from zookeeper, default is false", false, "false", &opts.useZooKeeperLag)
	toFlagStringsVar("zookeeper.server", "Address (hosts) of zookeeper server.", "localhost:2181", &opts.uriZookeeper)
	toFlagStringVar("kafka.labels", "Kafka cluster name", "", &opts.labels)
	toFlagStringVar("refresh.metadata", "Metadata refresh interval", "30s", &opts.metadataRefreshInterval)
	toFlagBoolVar("offset.show-all", "Whether show the offset/lag for all consumer group, otherwise, only show connected consumer groups, default is true", true, "true", &opts.offsetShowAll)
	toFlagBoolVar("concurrent.enable", "If true, all scrapes will trigger kafka operations otherwise, they will share results. WARN: This should be disabled on large clusters. Default is false", false, "false", &opts.allowConcurrent)
	toFlagIntVar("topic.workers", "Number of topic workers", 100, "100", &opts.topicWorkers)
	toFlagBoolVar("kafka.allow-auto-topic-creation", "If true, the broker may auto-create topics that we requested which do not already exist, default is false.", false, "false", &opts.allowAutoTopicCreation)
	toFlagIntVar("verbosity", "Verbosity log level", 0, "0", &opts.verbosityLogLevel)

	plConfig := plog.Config{}
	plogflag.AddFlags(kingpin.CommandLine, &plConfig)
	kingpin.Version(version.Print("kafka_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	labels := make(map[string]string)

	// Protect against empty labels
	if opts.labels != "" {
		for _, label := range strings.Split(opts.labels, ",") {
			splitted := strings.Split(label, "=")
			if len(splitted) >= 2 {
				labels[splitted[0]] = splitted[1]
			}
		}
	}

	Setup(*listenAddress, *metricsPath, *topicFilter, *groupFilter, *logSarama, opts, labels)
}

func Setup(
	listenAddress string,
	metricsPath string,
	topicFilter string,
	groupFilter string,
	logSarama bool,
	opts KafkaOpts,
	labels map[string]string,
) {
	klog.InitFlags(flag.CommandLine)
	if err := flag.Set("logtostderr", "true"); err != nil {
		klog.Errorf("Error on setting logtostderr to true: %v", err)
	}
	err := flag.Set("v", strconv.Itoa(opts.verbosityLogLevel))
	if err != nil {
		klog.Errorf("Error on setting v to %v: %v", strconv.Itoa(opts.verbosityLogLevel), err)
	}
	defer klog.Flush()

	klog.V(INFO).Infoln("Starting kafka_exporter", version.Info())
	klog.V(DEBUG).Infoln("Build context", version.BuildContext())

	if logSarama {
		sarama.Logger = log.New(os.Stdout, "[sarama] ", log.LstdFlags)
	}

	exporter, err := NewExporter(opts, topicFilter, groupFilter)
	if err != nil {
		klog.Fatalln(err)
	}
	defer exporter.client.Close()
	prometheus.MustRegister(exporter)

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
			klog.Error("Error handle / request", err)
		}
	})
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		// need more specific sarama check
		_, err := w.Write([]byte("ok"))
		if err != nil {
			klog.Error("Error handle /healthz request", err)
		}
	})

	if opts.serverUseTLS {
		klog.V(INFO).Infoln("Listening on HTTPS", listenAddress)

		_, err := CanReadCertAndKey(opts.serverTlsCertFile, opts.serverTlsKeyFile)
		if err != nil {
			klog.Error("error reading server cert and key")
		}

		clientAuthType := tls.NoClientCert
		if opts.serverMutualAuthEnabled {
			clientAuthType = tls.RequireAndVerifyClientCert
		}

		certPool := x509.NewCertPool()
		if opts.serverTlsCAFile != "" {
			if caCert, err := ioutil.ReadFile(opts.serverTlsCAFile); err == nil {
				certPool.AppendCertsFromPEM(caCert)
			} else {
				klog.Error("error reading server ca")
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
		klog.Fatal(server.ListenAndServeTLS(opts.serverTlsCertFile, opts.serverTlsKeyFile))
	} else {
		klog.V(INFO).Infoln("Listening on HTTP", listenAddress)
		klog.Fatal(http.ListenAndServe(listenAddress, nil))
	}
}
