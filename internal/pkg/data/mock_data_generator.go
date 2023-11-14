package data

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	aero "github.com/aerospike/aerospike-client-go/v6"
)

/*
Dummy Raw Metrics, copied from local Aerospike Server
returns static test data copied from running an Aerospike Server with test namespaces, sets, sindex, jobs, latencies etc.,
we need to update this data for each release to reflect the new metrics, contexts etc.,
this data is passed to the watcher and expected output is also generated
once we have output from watcher-implementations ( like watcher_namespaces.go, watcher_node_stats.go)

	this output is compated with the expected results generated by Test-Cases
*/

var MOCK_DATA_FILE = "tests/mock_test_data.txt"

func (mas MockAerospikeServer) RequestInfo(infokeys []string) (map[string]string, error) {
	fmt.Println("RequestInfo... ", infokeys)

	return mas.fetchRequestInfoFromFile(infokeys), nil
}

func (mas MockAerospikeServer) FetchUsersDetails() (bool, []*aero.UserRoles, error) {

	var aero_users []*aero.UserRoles

	users := mas.getUsersDetails("")

	user_keys := strings.Split(users, ";")

	// fmt.Println("users string: ", users)
	// fmt.Println(user_keys)

	for _, l_user_key := range user_keys {
		if len(l_user_key) > 0 {
			l_aero_user := mas.constructAeroUserRolesObject(l_user_key)
			aero_users = append(aero_users, l_aero_user)
		}
	}

	// fmt.Println(aero_users)

	return true, aero_users, nil
}

// Mock Data Provider related code, Inherits DataProvider interface
type MockAerospikeServer struct {
	Namespaces_stats []string
	Sets_stats       []string
	Xdr_stats        []string
	Node_stats       []string
	Latencies_stats  []string
	Sindex_stats     []string

	Build               []string
	Cluster_name        []string
	Service_clear_std   []string
	Namespaces          []string
	Sindexes            []string
	XdrContext          []string
	Users               []string
	Passone_output_str  string
	Passone_outputs_map map[string]string
}

// read mock test data from a file
var Is_Mock_Initialized = 0

const (
	MOCK_IK_BUILD                      string = "build"
	MOCK_IK_CLUSTER_NAME               string = "cluster-name"
	MOCK_IK_SERVICE_CLEAR_STD          string = "service-clear-std"
	MOCK_IK_NODE_STATISTICS            string = "statistics"
	MOCK_IK_GET_CONFIG_CONTEXT_SERVICE string = "get-config:context=service"
	MOCK_IK_SETS                       string = "sets"
	MOCK_IK_NAMESPACES                 string = "namespaces"
	MOCK_IK_SINDEX                     string = "sindex"
	MOCK_IK_NAMESPACE_SLASH            string = "namespace/"
	MOCK_IK_SINDEX_SLASH               string = "sindex/"
	MOCK_IK_XDR_CONFIG                 string = "get-config:context=xdr"
	MOCK_IK_XDR_STATS_DC               string = "get-stats:context=xdr;dc"
	MOCK_IK_XDR_CONFIG_DC              string = "get-config:context=xdr;dc"
	MOCK_IK_LATENCIES                  string = "latencies"
)

func (md *MockAerospikeServer) Initialize() {
	filePath := MOCK_DATA_FILE

	if _, err := os.Stat(filePath); err != nil {
		fmt.Println(filePath, " - File does not exist, may be running in Unit-test mode ")
		filePath = "../../../" + filePath
	}

	md.internalInitialize(filePath)
}

func (md *MockAerospikeServer) internalInitialize(filePath string) {

	// avoid multiple initializations
	if Is_Mock_Initialized == 1 {
		// fmt.Println("Mock data provider already Initialized: ")
		return
	}
	fmt.Println("************************************************************************************************************")
	fmt.Println("*                                                                                                          *")
	fmt.Println("*                                                                                                          *")
	fmt.Println("*             Mock Aerospike Server is enabled, going to use mock data                                     *")
	fmt.Println("*             Initializing mock-data-provider-data from file", MOCK_DATA_FILE, " ")
	fmt.Println("*                                                                                                          *")
	fmt.Println("*                                                                                                          *")
	fmt.Println("************************************************************************************************************")

	// Mark as initialized
	Is_Mock_Initialized = 1

	readFile, err := os.Open(filePath)

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Loading mock-data provider data from file :", filePath)

	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	var fileLines []string

	for fileScanner.Scan() {
		fileLines = append(fileLines, strings.TrimSpace(fileScanner.Text()))
	}

	readFile.Close()

	for _, line := range fileLines {
		if strings.HasPrefix(line, "#") && strings.HasPrefix(line, "//") {
			// ignore, comments
		} else if len(line) > 0 {
			if strings.HasPrefix(line, "namespace-") {
				md.Namespaces_stats = append(md.Namespaces_stats, line)
			} else if strings.HasPrefix(line, "set-stats") {
				md.Sets_stats = append(md.Sets_stats, line)
			} else if strings.HasPrefix(line, "latencies-stats") {
				md.Latencies_stats = append(md.Latencies_stats, line)
			} else if strings.HasPrefix(line, "node-") {
				md.Node_stats = append(md.Node_stats, line)
			} else if strings.HasPrefix(line, "xdr-") {
				md.Xdr_stats = append(md.Xdr_stats, line)
			} else if strings.HasPrefix(line, "user-stat") {
				md.Users = append(md.Users, line)
			} else if strings.HasPrefix(line, "get-config:context=xdr") { // Xdr configs
				md.XdrContext = append(md.XdrContext, line)
			} else if strings.HasPrefix(line, "sindex-stats:") {
				md.Sindex_stats = append(md.Sindex_stats, line)
			} else if strings.HasPrefix(line, "sindex:") {
				md.Sindexes = append(md.Sindexes, line)
			} else if strings.HasPrefix(line, "build") {
				md.Build = append(md.Build, line)
			} else if strings.HasPrefix(line, "service-clear-std") {
				md.Service_clear_std = append(md.Service_clear_std, line)
			} else if strings.HasPrefix(line, "cluster-name") {
				md.Cluster_name = append(md.Cluster_name, line)
			} else if strings.HasPrefix(line, "namespaces") {
				md.Namespaces = append(md.Namespaces, line)
			} else if strings.HasPrefix(line, "passone_output") {
				// passone_output:build:6.4.0.0-rc4 get-config:context=xdr:dcs=backup_dc_asdev20,backup_dc_asdev20_second;src-id=0;trace-sample=0 namespaces:test;bar_device;materials;ns_test_on_flash;test_on_shmem;bar_on_flash;pmkohl_on_device sindex:ns=test:indexname=test_sindex1:set=from_branch_2:bin=occurred:type=numeric:indextype=default:context=null:state=RW
				str := strings.ReplaceAll(line, "passone_output:", "")
				// store full string also
				md.Passone_output_str = str
				elements := strings.Split(md.Passone_output_str, " ")
				// reinitialize internal map
				md.Passone_outputs_map = make(map[string]string)
				for _, entry := range elements {

					colonIndex := strings.Index(entry, ":")
					// parts := strings.Split(entry, ":")
					key := entry[0:colonIndex]
					value := entry[colonIndex+1:]
					md.Passone_outputs_map[key] = value
				}
				// md.passone_outputs = splitAndRetrieveStats(str, ";")

			}

		}
	}
}

func (md *MockAerospikeServer) fetchRequestInfoFromFile(infokeys []string) map[string]string {
	var l_mock_data_map = make(map[string]string)

	for _, k := range infokeys {

		// fmt.Println("fetchRequestInfoFromFile(): processing key: ", k, "\t===> strings.HasPrefix(k, MOCK_IK_SINDEX_SLASH) ", strings.HasPrefix(k, MOCK_IK_SINDEX_SLASH))
		switch true {
		case strings.HasPrefix(k, MOCK_IK_BUILD):
			l_mock_data_map[k] = md.getBuild(k)
		case strings.HasPrefix(k, MOCK_IK_CLUSTER_NAME):
			l_mock_data_map[k] = md.getClusterName(k)
		case strings.HasPrefix(k, MOCK_IK_SERVICE_CLEAR_STD):
			l_mock_data_map[k] = md.getServiceClearStd(k)
		case strings.HasPrefix(k, MOCK_IK_NAMESPACES):
			l_mock_data_map[k] = md.getNamespaces(k)
		case strings.HasPrefix(k, MOCK_IK_NAMESPACE_SLASH):
			l_mock_data_map[k] = md.getSingleNamespaceStats(k)
		case strings.HasPrefix(k, MOCK_IK_NODE_STATISTICS):
			l_mock_data_map[k] = md.getNodeStatistics(k)
		case strings.HasPrefix(k, MOCK_IK_GET_CONFIG_CONTEXT_SERVICE):
			l_mock_data_map[k] = md.getNodeStatistics(k)
		case strings.HasPrefix(k, MOCK_IK_SETS):
			l_mock_data_map[k] = md.getSetsStatistics(k)
		case (strings.HasPrefix(k, MOCK_IK_SINDEX) && !strings.Contains(k, "/")):
			l_mock_data_map[k] = md.getSindex(k)
		case strings.HasPrefix(k, MOCK_IK_SINDEX_SLASH):
			l_mock_data_map[k] = md.getSingleSindexStatistics(k)
		case strings.HasPrefix(k, MOCK_IK_XDR_CONFIG):
			l_mock_data_map[k] = md.getXdrConfigsContext(k)
		case strings.HasPrefix(k, MOCK_IK_XDR_CONFIG_DC):
			l_mock_data_map[k] = md.getSingleXdrKeys(k)
		case strings.HasPrefix(k, MOCK_IK_XDR_STATS_DC):
			l_mock_data_map[k] = md.getSingleXdrKeys(k)
		case strings.HasPrefix(k, MOCK_IK_LATENCIES):
			l_mock_data_map[k] = md.getLatenciesStats(k)
		}
	}
	// fmt.Println("requested keys : ", infokeys, "\n\t values returned: ", l_mock_data_map)
	return l_mock_data_map
}

func (md *MockAerospikeServer) getBuild(key string) string {
	return strings.Split(md.Build[0], "=")[1]
}

func (md *MockAerospikeServer) getClusterName(key string) string {
	return strings.Split(md.Cluster_name[0], "=")[1]
}

func (md *MockAerospikeServer) getServiceClearStd(key string) string {
	fmt.Println("\n\n****** getServiceClearStd() .... \t", strings.Split(md.Service_clear_std[0], "=")[1]+"\n")
	return strings.Split(md.Service_clear_std[0], "=")[1]
}

func (md *MockAerospikeServer) getNamespaces(key string) string {
	return strings.Split(md.Namespaces[0], ":")[1]
}

func (md *MockAerospikeServer) getSingleNamespaceStats(nsKey string) string {
	rawMetrics := ""

	ns := strings.Split(nsKey, "/")[1]
	// fmt.Println("reading metrics for namespaceKey: ", nsKey, " ---> and the namespace is : ", ns)

	// namespace
	for _, entry := range md.Namespaces_stats {
		elements := strings.Split(entry, ":")
		// format: namespace-stats:test:ns_cluster_size=1;effective_ ( 2nd element is the namespace name "test")
		if strings.Contains(entry, (":" + ns + ":")) {
			// key := "namespace/" + elements[1]
			rawMetrics = elements[2]
		}
	}

	return rawMetrics
}

func (md *MockAerospikeServer) getNodeStatistics(key string) string {
	rawMetrics := ""
	// node-stats & node-configs
	for _, entry := range md.Node_stats {

		// node-configs:<node-configs> & node-stats:<node-stats>
		elements := strings.Split(entry, ":")

		if strings.HasPrefix(key, "statistics") && strings.HasPrefix(elements[0], "node-stats") {
			// key := "statistics"
			rawMetrics = elements[1]
		} else if strings.HasPrefix(key, "get-config:context=service") && strings.HasPrefix(elements[0], "node-config") {
			// key := "get-config:context=service"
			rawMetrics = elements[1]
		}
	}

	return rawMetrics

}

func (md *MockAerospikeServer) getSetsStatistics(key string) string {
	rawMetrics := ""
	// node-stats & node-configs
	for _, entry := range md.Sets_stats {

		if strings.HasPrefix(key, "sets") && strings.HasPrefix(entry, "set-stats:") {
			// set-stats:<node-configs>
			elements := strings.Replace(entry, "set-stats:[", "", 1)
			elements = strings.Replace(elements, "]", "", 1)

			// key := "sets"
			rawMetrics = elements
		}
	}

	// fmt.Println(" ** getSetsStatistics() key: ", key, "\n\t values: ", rawMetrics)
	return rawMetrics

}

func (md *MockAerospikeServer) getSindex(key string) string {
	rawMetrics := ""
	// sindex
	for _, entry := range md.Sindexes {
		// fmt.Println("\tgetSindex() ... processing ", entry)
		if strings.HasPrefix(key, "sindex") && strings.HasPrefix(entry, "sindex:") {
			// set-stats:<node-configs>
			elements := strings.Replace(entry, "sindex:", "", 1)

			// key := "sets"
			rawMetrics = elements
		}
	}

	return rawMetrics

}

func (md *MockAerospikeServer) getSingleSindexStatistics(key string) string {
	rawMetrics := ""
	// sindex-stats
	for _, entry := range md.Sindex_stats {
		elements := strings.Split(entry, ":")

		if strings.HasPrefix(entry, "sindex-stats") && strings.HasPrefix(elements[1], key) {
			// key := "sindex"
			rawMetrics = elements[2]
		}
	}

	// fmt.Println(" ** getSingleSindexStatistics() key: ", key, "\n\t values: ", rawMetrics)
	return rawMetrics

}

func (md *MockAerospikeServer) getXdrConfigsContext(key string) string {
	rawMetrics := ""
	// sindex
	for _, entry := range md.XdrContext {
		// fmt.Println("\tgetSindex() ... processing ", entry)
		if strings.HasPrefix(key, "get-config:context=xdr") {
			// set-stats:<node-configs>
			elements := strings.Replace(entry, "get-config:context=xdr:", "", 1)

			// key := "sets"
			rawMetrics = elements
		}
	}

	return rawMetrics

}

func (md *MockAerospikeServer) getSingleXdrKeys(key string) string {
	rawMetrics := ""
	// xdr-stats and xdr-configs
	for _, entry := range md.Xdr_stats {
		elements := ""

		// fmt.Println("\n\t*** getSingleXdrKeys: ",
		// 	"\n\t key: ", key,
		// 	"\n\t entry: ", entry,
		// 	"\n\t strings.Contains(entry, key): ", strings.Contains(entry, key))
		if strings.HasPrefix(entry, "xdr") && strings.Contains(entry, key) {
			// key := "xdr-"
			elements = strings.Replace(entry, "xdr-", "", 1)
			elements = strings.Replace(elements, (key + ";"), "", 1)
			elements = strings.Replace(elements, (key + ":"), "", 1)

			rawMetrics = elements
			break
		}
	}

	// fmt.Println(" ** getSingleXdrKeys() key: ", key, "\n\t values: ", rawMetrics)
	return rawMetrics
}

func (md *MockAerospikeServer) getLatenciesStats(latenciesKey string) string {
	rawMetrics := ""

	// Latencies
	for _, entry := range md.Latencies_stats {
		// format: latencies-stats:
		if strings.Contains(entry, ("latencies-stats:")) {
			// key := "latencies-stats:"
			elements := strings.Replace(entry, "latencies-stats:", "", 1)
			rawMetrics = elements
		}
	}

	return rawMetrics
}

func (md *MockAerospikeServer) getUsersDetails(key string) string {
	rawMetrics := ""
	// users
	elements := md.Users[0]
	elements = strings.Replace(elements, "user-stats:", "", 1)
	rawMetrics = elements

	return rawMetrics

}

func (md *MockAerospikeServer) constructAeroUserRolesObject(key string) *aero.UserRoles {

	// fmt.Println("\nprocessing user-key :", key)
	var err error
	tmp_user_role := &aero.UserRoles{}

	elements := strings.Split(key, ":")

	// user
	tmp_user_role.User = strings.Split(elements[0], "=")[1]

	// roles assigned
	tmp_user_role.Roles = strings.Split(strings.Split(elements[1], "=")[1], "-")

	// conns-in-use
	s_conns_in_user := strings.Split(elements[2], "=")[1]
	tmp_user_role.ConnsInUse = 0
	if s_conns_in_user != "" {
		tmp_user_role.ConnsInUse, err = strconv.Atoi(strings.Split(elements[2], "=")[1])
	}

	// read-info
	if len(elements) > 2 {
		// read-info=1-2-3-4
		l_read_info := strings.Split(strings.Split(elements[3], "=")[1], "-")
		// fmt.Println("read-infos: ", elements[3])
		l_int_read_info := []int{0, 0, 0, 0}
		for i := 0; i < len(l_read_info); i++ {
			l_int_read_info[i] = 0
			if l_read_info[i] != "" {
				l_int_read_info[i], err = strconv.Atoi(l_read_info[i])
			}
		}
		tmp_user_role.ReadInfo = l_int_read_info

		// write-info=11-12-13-14
		l_write_info := strings.Split(strings.Split(elements[4], "=")[1], "-")
		l_int_write_info := []int{0, 0, 0, 0}
		// fmt.Println("write-infos: ", elements[4])
		for i := 0; i < len(l_write_info); i++ {
			l_int_write_info[i] = 0
			if l_read_info[i] != "" {
				l_int_write_info[i], err = strconv.Atoi(l_write_info[i])
			}
		}
		tmp_user_role.WriteInfo = l_int_write_info

		if err != nil {
			fmt.Println("error while convering user-stat values: ", err)
		}

	}

	return tmp_user_role
}
