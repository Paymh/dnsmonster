package main

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/namsral/flag"

	log "github.com/sirupsen/logrus"
)

// This is needed for supporting multiple inputs for Splunk output
type splunkOutputEndpointList []string

func (i *splunkOutputEndpointList) String() string {
	return strings.Join(*i, " ")
}
func (i *splunkOutputEndpointList) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var splunkOutputEndpoints splunkOutputEndpointList

var fs = flag.NewFlagSetWithEnvPrefix(os.Args[0], "DNSMONSTER", 0)
var devName = fs.String("devName", "", "Device used to capture")
var pcapFile = fs.String("pcapFile", "", "Pcap filename to run")
var dnstapSocket = fs.String("dnstapSocket", "", "dnstrap socket path. Example: unix:///tmp/dnstap.sock, tcp://127.0.0.1:8080")

var config = fs.String(flag.DefaultConfigFlagname, "", "path to config file")
var filter = fs.String("filter", "((ip and (ip[9] == 6 or ip[9] == 17)) or (ip6 and (ip6[6] == 17 or ip6[6] == 6 or ip6[6] == 44)))", "BPF filter applied to the packet stream. If port is selected, the packets will not be defragged.")
var port = fs.Uint("port", 53, "Port selected to filter packets")
var gcTime = fs.Duration("gcTime", 10*time.Second, "Garbage Collection interval for tcp assembly and ip defragmentation")
var clickhouseAddress = fs.String("clickhouseAddress", "localhost:9000", "Address of the clickhouse database to save the results")
var clickhouseDelay = fs.Duration("clickhouseDelay", 1*time.Second, "Interval between sending results to ClickHouse")
var clickhouseDebug = fs.Bool("clickhouseDebug", false, "Debug Clickhouse connection")
var clickhouseOutputType = fs.Uint("clickhouseOutputType", 0, "What should be written to clickhouse. options: 0: none, 1: all, 2: apply skipdomains logic, 3: apply allowdomains logic, 4: apply both skip and allow domains logic")
var clickhouseBatchSize = fs.Uint("clickhouseBatchSize", 100000, "Minimun capacity of the cache array used to send data to clickhouse. Set close to the queries per second received to prevent allocations")
var captureStatsDelay = fs.Duration("captureStatsDelay", time.Second, "Duration to calculate interface stats")
var printStatsDelay = fs.Duration("printStatsDelay", time.Second*10, "Duration to print capture and database stats")
var maskSize = fs.Int("maskSize", 32, "Mask source IPs by bits. 32 means all the bits of IP is saved in DB")
var serverName = fs.String("serverName", "default", "Name of the server used to index the metrics.")
var sampleRatio = fs.String("sampleRatio", "1:1", "Capture Sampling by a:b. eg sampleRatio of 1:100 will process 1 percent of the incoming packets")
var saveFullQuery = fs.Bool("saveFullQuery", false, "Save full packet query and response in JSON format. Will respect maskSize")
var packetHandlerCount = fs.Uint("packetHandlers", 1, "Number of routines used to handle received packets")
var tcpHandlerCount = fs.Uint("tcpHandlers", 1, "Number of routines used to handle tcp assembly")
var useAfpacket = fs.Bool("useAfpacket", false, "Use AFPacket for live captures. Supported on Linux 3.0+ only")
var afpacketBuffersizeMb = fs.Uint("afpacketBuffersizeMb", 64, "Afpacket Buffersize in MB")
var packetChannelSize = fs.Uint("packetHandlerChannelSize", 100000, "Size of the packet handler channel")
var tcpAssemblyChannelSize = fs.Uint("tcpAssemblyChannelSize", 1000, "Size of the tcp assembler")
var tcpResultChannelSize = fs.Uint("tcpResultChannelSize", 1000, "Size of the tcp result channel")
var resultChannelSize = fs.Uint("resultChannelSize", 100000, "Size of the result processor channel size")
var logLevel = fs.Uint("logLevel", 3, "Set debug Log level, 0:PANIC, 1:ERROR, 2:WARN, 3:INFO, 4:DEBUG")
var defraggerChannelSize = fs.Uint("defraggerChannelSize", 500, "Size of the channel to send packets to be defragged")
var defraggerChannelReturnSize = fs.Uint("defraggerChannelReturnSize", 500, "Size of the channel where the defragged packets are returned")
var cpuprofile = fs.String("cpuprofile", "", "write cpu profile to file")
var memprofile = fs.String("memprofile", "", "write memory profile to file")
var gomaxprocs = fs.Int("gomaxprocs", -1, "GOMAXPROCS variable")
var skipTLSVerification = fs.Bool("skipTLSVerification", false, "Skip TLS verification when making HTTPS connections")
var packetLimit = fs.Int("packetLimit", 0, "Limit of packets logged to clickhouse every iteration. Default 0 (disabled)")
var skipDomainsFile = fs.String("skipDomainsFile", "", "Skip outputing domains matching items in the CSV file path. Can accept a URL (http:// or https://) or path")
var skipDomainsRefreshInterval = fs.Duration("skipDomainsRefreshInterval", 60*time.Second, "Hot-Reload skipDomainsFile interval")
var skipDomainsFileType = fs.String("skipDomainsFileType", "csv", "skipDomainsFile type. Options: csv and hashtable. Hashtable is ONLY fqdn, csv can support fqdn, prefix and suffix logic but it's much slower")
var allowDomainsFile = fs.String("allowDomainsFile", "", "Allow Domains logic input file. Can accept a URL (http:// or https://) or path")
var allowDomainsRefreshInterval = fs.Duration("allowDomainsRefreshInterval", 60*time.Second, "Hot-Reload allowDomainsFile file interval")
var allowDomainsFileType = fs.String("allowDomainsFileType", "csv", "allowDomainsFile type. Options: csv and hashtable. Hashtable is ONLY fqdn, csv can support fqdn, prefix and suffix logic but it's much slower")
var dnstapPermission = fs.String("dnstapPermission", "755", "Set the dnstap socket permission, only applicable when unix:// is used")
var fileOutputType = fs.Uint("fileOutputType", 0, "What should be written to file. options: 0: none, 1: all, 2: apply skipdomains logic, 3: apply allowdomains logic, 4: apply both skip and allow domains logic")
var fileOutputPath = fs.String("fileOutputPath", "", "Path to output file. Used if fileOutputType is not none")
var stdoutOutputType = fs.Uint("stdoutOutputType", 0, "What should be written to stdout. options: 0: none, 1: all, 2: apply skipdomains logic, 3: apply allowdomains logic, 4: apply both skip and allow domains logic")
var syslogOutputType = fs.Uint("syslogOutputType", 0, "What should be written to Syslog server. options: 0: none, 1: all, 2: apply skipdomains logic, 3: apply allowdomains logic, 4: apply both skip and allow domains logic")
var syslogOutputEndpoint = fs.String("syslogOutputEndpoint", "", "Syslog endpoint address, example: udp://127.0.0.1:514, tcp://127.0.0.1:514. Used if syslogOutputType is not none")
var kafkaOutputType = fs.Uint("kafkaOutputType", 0, "What should be written to kafka. options: 0: none, 1: all, 2: apply skipdomains logic, 3: apply allowdomains logic, 4: apply both skip and allow domains logic")
var kafkaOutputBroker = fs.String("kafkaOutputBroker", "", "kafka broker address, example: 127.0.0.1:9092. Used if kafkaOutputType is not none")
var kafkaOutputTopic = fs.String("kafkaOutputTopic", "dnsmonster", "Kafka topic for logging")
var kafkaBatchSize = fs.Uint("kafkaBatchSize", 1000, "Minimun capacity of the cache array used to send data to Kafka")
var kafkaBatchDelay = fs.Duration("kafkaBatchDelay", 1*time.Second, "Interval between sending results to Kafka if Batch size is not filled")
var elasticOutputType = fs.Uint("elasticOutputType", 0, "What should be written to elastic. options: 0: none, 1: all, 2: apply skipdomains logic, 3: apply allowdomains logic, 4: apply both skip and allow domains logic")
var elasticOutputEndpoint = fs.String("elasticOutputEndpoint", "", "elastic endpoint address, example: http://127.0.0.1:9200. Used if elasticOutputType is not none")
var elasticOutputIndex = fs.String("elasticOutputIndex", "default", "elastic index")
var elasticBatchSize = fs.Uint("elasticBatchSize", 1000, "Send data to Elastic in batch sizes")
var elasticBatchDelay = fs.Duration("elasticBatchDelay", 1*time.Second, "Interval between sending results to Elastic if Batch size is not filled")
var splunkOutputType = fs.Uint("splunkOutputType", 0, "What should be written to HEC. options: 0: none, 1: all, 2: apply skipdomains logic, 3: apply allowdomains logic, 4: apply both skip and allow domains logic")
var splunkOutputToken = fs.String("splunkOutputToken", "00000000-0000-0000-0000-000000000000", "Splunk HEC Token")
var splunkOutputIndex = fs.String("splunkOutputIndex", "temp", "Splunk Output Index")
var splunkOutputSource = fs.String("splunkOutputSource", "dnsmonster", "Splunk Output Source")
var splunkOutputSourceType = fs.String("splunkOutputSourceType", "json", "Splunk Output Sourcetype")
var splunkBatchSize = fs.Uint("splunkBatchSize", 1000, "Send data to HEC in batch sizes")
var splunkBatchDelay = fs.Duration("splunkBatchDelay", 1*time.Second, "Interval between sending results to HEC if Batch size is not filled")

var version = fs.Bool("version", false, "show version and exit")

func checkFlags() {

	fs.Var(&splunkOutputEndpoints, "splunkOutputEndpoint", "HEC endpoint address, example: http://127.0.0.1:8088. Used if splunkOutputType is not none")
	log.Info("Parsing flags")
	err := fs.Parse(os.Args[1:])
	errorHandler(err)

	// defualt logging to warning
	var lvl log.Level = log.WarnLevel
	switch *logLevel {
	case 0:
		lvl = log.PanicLevel
	case 1:
		lvl = log.ErrorLevel
	case 2:
		lvl = log.WarnLevel
	case 3:
		lvl = log.InfoLevel
	case 4:
		lvl = log.DebugLevel
	}
	log.SetLevel(lvl)

	if *version {
		log.Fatalln("dnsmonster version:", releaseVersion)
	}

	//TODO: log format needs to be a configurable parameter
	log.SetFormatter(&log.JSONFormatter{})

	if *skipDomainsFile != "" {
		log.Info("skipDomainsFile is provided")
		// check to see if the file provided exists
		// commented because this now can be either filepath or URL, TODO
		// if _, err := os.Stat(*skipDomainsFile); err != nil {
		// 	log.Fatal("error in finding SkipDomains file. You must provide a path to an existing filename")
		// }
		if *skipDomainsFileType != "csv" && *skipDomainsFileType != "hashtable" {
			log.Fatal("skipDomainsFileType must be either csv or hashtable")
		}
		if *skipDomainsFileType == "hashtable" {
			skipDomainMapBool = true
		}
	}

	if *allowDomainsFile != "" {
		log.Info("allowDomainsFile is provided")
		// check to see if the file provided exists
		// commented because this now can be either filepath or URL, TODO
		// if _, err := os.Stat(*allowDomainsFile); err != nil {
		// 	log.Fatal("error in finding allowDomainsFile. You must provide a path to an existing filename")
		// }
		if *allowDomainsFileType != "csv" && *allowDomainsFileType != "hashtable" {
			log.Fatal("allowDomainsFileType must be either csv or hashtable")
		}
		if *allowDomainsFileType == "hashtable" {
			allowDomainMapBool = true
		}
	}

	if *stdoutOutputType >= 5 {
		log.Fatal("stdoutOutputType must be one of 0, 1, 2, 3 or 4")
	}
	if *fileOutputType >= 5 {
		log.Fatal("fileOutputType must be one of 0, 1, 2, 3 or 4")
	} else if *fileOutputType > 0 {
		if *fileOutputPath == "" {
			log.Fatal("fileOutputType is set but fileOutputPath is not provided. Exiting")
		}
	}
	if *clickhouseOutputType >= 5 {
		log.Fatal("clickhouseOutputType must be one of 0, 1, 2, 3 or 4")
	}
	if *kafkaOutputType >= 5 {
		log.Fatal("kafkaOutputType must be one of 0, 1, 2, 3 or 4")
	}
	if *elasticOutputType >= 5 {
		log.Fatal("elasticOutputType must be one of 0, 1, 2, 3 or 4")
	}
	if *port > 65535 {
		log.Fatal("-port must be between 1 and 65535")
	}
	if *maskSize > 32 || *maskSize < 0 {
		log.Fatal("-maskSize must be between 0 and 32")
	}
	if *devName == "" && *pcapFile == "" && *dnstapSocket == "" {
		log.Fatal("one of -devName, -pcapFile or -dnstapSocket is required")
	}

	if *devName != "" {
		if *pcapFile != "" || *dnstapSocket != "" {
			log.Fatal("You must set only -devName, -pcapFile or -dnstapSocket")
		}
	} else {
		if *pcapFile != "" && *dnstapSocket != "" {
			log.Fatal("You must set only -devName, -pcapFile or -dnstapSocket")
		}
	}

	if *dnstapSocket != "" {
		if !strings.HasPrefix(*dnstapSocket, "unix://") && !strings.HasPrefix(*dnstapSocket, "tcp://") {
			log.Fatal("You must provide a unix:// or tcp:// socket for dnstap")
		}
	}

	if *packetLimit < 0 {
		log.Fatal("-packetLimit must be equal or greather than 0")
	}

	ratioNumbers := strings.Split(*sampleRatio, ":")
	if len(ratioNumbers) != 2 {
		log.Fatal("wrong -sampleRatio syntax")
	}
	var errA error
	var errB error
	ratioA, errA = strconv.Atoi(ratioNumbers[0])
	ratioB, errB = strconv.Atoi(ratioNumbers[1])
	if errA != nil || errB != nil || ratioA > ratioB {
		log.Fatal("wrong -sampleRatio syntax")
	}

}
