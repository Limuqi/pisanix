// Copyright 2022 SphereEx Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package proxy

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/database-mesh/golang-sdk/client"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var vdb = client.VirtualDatabase{
	ObjectMeta: metav1.ObjectMeta{
		Name: "catalogue",
	},
	Spec: client.VirtualDatabaseSpec{
		Services: []client.VirtualDatabaseService{
			{
				DatabaseService: client.DatabaseService{
					DatabaseMySQL: &client.DatabaseMySQL{
						Host:          "127.0.0.1",
						Port:          3306,
						DB:            "socksdb",
						User:          "root",
						Password:      "fake_password",
						ServerVersion: "5.7.37",
						PoolSize:      3,
					},
				},
				Name:            "catalogue",
				TrafficStrategy: "catalogue",
			},
		},
	},
}

var dbep = client.DatabaseEndpoint{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "catalogue",
		Namespace: "demotest",
		Annotations: map[string]string{
			DatabaseEndpointRoleKey: ReadWriteSplittingRoleReadWrite,
		},
		Labels: map[string]string{
			"source": "catalogue",
		},
	},
	Spec: client.DatabaseEndpointSpec{
		Database: client.Database{
			MySQL: &client.MySQL{
				DB:       "socksdb",
				Host:     "catalogue-db.demotest",
				Password: "fake_password",
				Port:     3306,
				User:     "root",
			},
		},
	},
}

var cbs = []client.CircuitBreak{
	{
		Regex: []string{
			"^select",
		},
	},
}

var ccs = []client.ConcurrencyControl{
	{
		Regex: []string{
			"^insert",
		},
		Duration:       10 * time.Second,
		MaxConcurrency: 10,
	},
}

var tsSimpleLoadBalance = client.TrafficStrategy{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "catalogue",
		Namespace: "demotest",
	},
	Spec: client.TrafficStrategySpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"source": "catalogue",
			},
		},
		CircuitBreaks:       cbs,
		ConcurrencyControls: ccs,
		LoadBalance: &client.LoadBalance{
			SimpleLoadBalance: &client.SimpleLoadBalance{
				Kind: client.LoadBalanceAlgorithmRoundRobin,
			},
		},
	},
}
var tsReadWriteSplittingStatic = client.TrafficStrategy{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "catalogue",
		Namespace: "demotest",
	},
	Spec: client.TrafficStrategySpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"source": "catalogue",
			},
		},
		CircuitBreaks:       cbs,
		ConcurrencyControls: ccs,
		LoadBalance: &client.LoadBalance{
			ReadWriteSplitting: &client.ReadWriteSplitting{
				Static: &client.ReadWriteSplittingStatic{
					DefaultTarget: "readwrite",
					Rules: []client.ReadWriteSplittingRule{
						{
							Name:          "write-rule",
							Regex:         []string{"^insert"},
							Target:        "readwrite",
							Type:          "regex",
							AlgorithmName: "roundrobin",
						},
						{
							Name:          "read-rule",
							Regex:         []string{"^select"},
							Target:        "read",
							Type:          "regex",
							AlgorithmName: "roundrobin",
						},
					},
				},
			},
		},
	},
}
var tsReadWriteSplttingDynamic = client.TrafficStrategy{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "catalogue",
		Namespace: "demotest",
	},
	Spec: client.TrafficStrategySpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"source": "catalogue",
			},
		},
		CircuitBreaks:       cbs,
		ConcurrencyControls: ccs,
		LoadBalance: &client.LoadBalance{
			ReadWriteSplitting: &client.ReadWriteSplitting{
				Dynamic: &client.ReadWriteSplittingDynamic{
					DefaultTarget: "readwrite",
					Rules: []client.ReadWriteSplittingRule{
						{
							Name:          "write-rule",
							Regex:         []string{"^insert"},
							Target:        "readwrite",
							Type:          "regex",
							AlgorithmName: "roundrobin",
						},
						{
							Name:          "read-rule",
							Regex:         []string{"^select"},
							Target:        "read",
							Type:          "regex",
							AlgorithmName: "roundrobin",
						},
					},
					Discovery: client.ReadWriteDiscovery{
						MasterHighAvailability: &client.MasterHighAvailability{
							User:            "monitor",
							Password:        "monitor",
							MonitorInterval: 1000,
							ConnectionProbe: &client.ConnectionProbe{
								Probe: &client.Probe{
									PeriodMilliseconds:  2000,
									FailureThreshold:    3,
									TimeoutMilliseconds: 200,
								},
							},
							PingProbe: &client.PingProbe{
								Probe: &client.Probe{
									PeriodMilliseconds:  1000,
									TimeoutMilliseconds: 100,
									FailureThreshold:    3,
								},
							},
							ReplicationLagProbe: &client.ReplicationLagProbe{
								Probe: &client.Probe{
									PeriodMilliseconds:  1000,
									TimeoutMilliseconds: 3,
									FailureThreshold:    3,
								},
								MaxReplicationLag: 3,
							},
							ReadOnlyProbe: &client.ReadOnlyProbe{
								Probe: &client.Probe{
									PeriodMilliseconds:  1000,
									TimeoutMilliseconds: 3,
									FailureThreshold:    3,
								},
							},
						},
					},
				},
			},
		},
	},
}

var expectedProxy = &Proxy{
	Name:          "catalogue",
	BackendType:   "mysql",
	DB:            "socksdb",
	User:          "root",
	Password:      "fake_password",
	ServerVersion: "5.7.37",
	PoolSize:      3,
	ListenAddr:    "127.0.0.1:3306",
	SimpleLoadBalance: &SimpleLoadBalance{
		BalancerType: "roundrobin",
		Nodes:        []string{"catalogue"},
	},
	ReadWriteSplitting: &ReadWriteSplitting{
		Static: &ReadWriteSplittingStatic{
			DefaultTarget: "readwrite",
			Rules: []ReadWriteSplittingRule{
				{
					Name:          "write-rule",
					Regex:         []string{"^insert"},
					Target:        "readwrite",
					Type:          "regex",
					AlgorithmName: "roundrobin",
				},
				{
					Name:          "read-rule",
					Regex:         []string{"^select"},
					Target:        "read",
					Type:          "regex",
					AlgorithmName: "roundrobin",
				},
			},
		},
		Dynamic: &ReadWriteSplittingDynamic{
			DefaultTarget: "readwrite",
			Rules: []ReadWriteSplittingRule{
				{
					Name:          "write-rule",
					Regex:         []string{"^insert"},
					Target:        "readwrite",
					Type:          "regex",
					AlgorithmName: "roundrobin",
				},
				{
					Name:          "read-rule",
					Regex:         []string{"^select"},
					Target:        "read",
					Type:          "regex",
					AlgorithmName: "roundrobin",
				},
			},
			Discovery: ReadWriteDiscovery{
				MasterHighAvailablity: &MasterHighAvailablity{
					Type:                      "mha",
					User:                      "monitor",
					Password:                  "monitor",
					MonitorInterval:           1000,
					ConnectInterval:           2000,
					ConnectMaxFailures:        3,
					ConnectTimeout:            200,
					PingInterval:              1000,
					PingTimeout:               100,
					PingMaxFailures:           3,
					ReplicationLagInterval:    1000,
					ReplicationLagTimeout:     3,
					ReplicationLagMaxFailures: 3,
					MaxReplicationLag:         3,
					ReadOnlyInterval:          1000,
					ReadOnlyTimeout:           3,
					ReadOnlyMaxFailures:       3,
				},
			},
		},
	},
	Plugin: &Plugin{
		CircuitBreaks: []CircuitBreak{
			{
				Regex: []string{
					"^select",
				},
			},
		},
		ConcurrencyControls: []ConcurrencyControl{
			{
				Regex: []string{
					"^insert",
				},
				Duration:       10 * time.Second,
				MaxConcurrency: 10,
			},
		},
	},
}

func Test_ProxyBuilder(t *testing.T) {
	builders := []*ProxyBuilder{
		{
			VirtualDatabaseService: vdb.Spec.Services[0],
			TrafficStrategy:        tsSimpleLoadBalance,
			DatabaseEndpoints:      []client.DatabaseEndpoint{dbep},
		},
		{
			VirtualDatabaseService: vdb.Spec.Services[0],
			TrafficStrategy:        tsReadWriteSplittingStatic,
			DatabaseEndpoints:      []client.DatabaseEndpoint{dbep},
		},
		{
			VirtualDatabaseService: vdb.Spec.Services[0],
			TrafficStrategy:        tsReadWriteSplttingDynamic,
			DatabaseEndpoints:      []client.DatabaseEndpoint{dbep},
		},
	}

	for _, b := range builders {
		actual := b.Build()
		assertProxy(t, expectedProxy, actual, "proxy should be correct")
	}
}

func assertProxy(t *testing.T, exp, act *Proxy, msg ...interface{}) bool {
	return assert.Equal(t, exp.BackendType, act.BackendType, "backendType should be equal") &&
		assert.Equal(t, exp.DB, act.DB, "db should be equal") &&
		assert.Equal(t, exp.ListenAddr, act.ListenAddr, "listenAddr should be equal") &&
		assert.Equal(t, exp.Name, act.Name, "name should be equal") &&
		assert.Equal(t, exp.Password, act.Password, "password should be equal") &&
		assert.Equal(t, exp.PoolSize, act.PoolSize, "poolSize should be equal") &&
		assert.Equal(t, exp.User, act.User, "user should be equal") &&
		assert.Equal(t, exp.ServerVersion, act.ServerVersion, "serverVersion should be equal") &&
		assertSimpleLoadBalance(t, exp.SimpleLoadBalance, act.SimpleLoadBalance, "simpleLoadBalance should be equal") &&
		assertReadWriteSplitting(t, exp.ReadWriteSplitting, act.ReadWriteSplitting, "readWriteSplitting should be equal") &&
		assertPlugin(t, exp.Plugin, act.Plugin, "plugin should be equal")
}

func assertSimpleLoadBalance(t *testing.T, act, exp *SimpleLoadBalance, msg ...interface{}) bool {
	if act != nil && exp != nil {
		return assert.Equal(t, act.BalancerType, exp.BalancerType, "balancerType should be equal") &&
			assert.ElementsMatch(t, act.Nodes, exp.Nodes, "nodes should be equal")

	}
	return true
}

func assertReadWriteSplitting(t *testing.T, act, exp *ReadWriteSplitting, msg ...interface{}) bool {
	if act != nil && exp != nil {
		return assertReadWriteSplittingStatic(t, act.Static, exp.Static, "readWriteSplittingStatic should be equal") &&
			assertReadWriteSplittingDynamic(t, act.Dynamic, exp.Dynamic, "readWriteSplittingDynamic should be equal")
	}
	return true
}

func assertReadWriteSplittingStatic(t *testing.T, act, exp *ReadWriteSplittingStatic, msg ...interface{}) bool {
	if act != nil && exp != nil {
		return assert.Equal(t, act.DefaultTarget, exp.DefaultTarget, "defaultTarget should be equal") &&
			//TODO: check if this is valid
			assert.Equal(t, act.Rules, exp.Rules, "rules should be equal")
	}
	return true
}

func assertReadWriteSplittingDynamic(t *testing.T, act, exp *ReadWriteSplittingDynamic, msg ...interface{}) bool {
	if act != nil && exp != nil {
		return assert.Equal(t, act.DefaultTarget, exp.DefaultTarget, "defaultType should be equal") &&
			assert.Equal(t, act.Rules, exp.Rules, "rules should be equal") &&
			assertReadWriteDiscovery(t, act.Discovery, exp.Discovery, "discovery should be equal")

	}
	return true
}

func assertReadWriteDiscovery(t *testing.T, act, exp ReadWriteDiscovery, msg ...interface{}) bool {
	return assert.Equal(t, act.Type, exp.Type, "type should be equal") &&
		assert.Equal(t, act.User, exp.User, "user should be equal") &&
		assert.Equal(t, act.Password, exp.Password, "password should be equal") &&
		assert.Equal(t, act.MonitorInterval, exp.MonitorInterval, "monitorInterval should be equal") &&
		assert.Equal(t, act.ConnectInterval, exp.ConnectInterval, "connectInterval should be equal") &&
		assert.Equal(t, act.ConnectTimeout, exp.ConnectTimeout, "connectTimeout should be equal") &&
		assert.Equal(t, act.ConnectMaxFailures, exp.ConnectMaxFailures, "connectMaxFailures should be equal") &&
		assert.Equal(t, act.PingInterval, exp.PingInterval, "pingInterval should be equal") &&
		assert.Equal(t, act.PingTimeout, exp.PingTimeout, "pingTimeout should be equal") &&
		assert.Equal(t, act.PingMaxFailures, exp.PingMaxFailures, "pingMaxFailures should be equal") &&
		assert.Equal(t, act.ReplicationLagInterval, exp.ReplicationLagInterval, "replicationLagInterval should be equal") &&
		assert.Equal(t, act.ReplicationLagTimeout, exp.ReplicationLagTimeout, "replicationLagTimeout should be equal") &&
		assert.Equal(t, act.ReplicationLagMaxFailures, exp.ReplicationLagMaxFailures, "replicationLagMaxFailures should be equal") &&
		assert.Equal(t, act.MaxReplicationLag, exp.MaxReplicationLag, "maxReplicationLagLag should be equal") &&
		assert.Equal(t, act.ReadOnlyInterval, exp.ReadOnlyInterval, "readOnlyInterval should be equal") &&
		assert.Equal(t, act.ReadOnlyTimeout, exp.ReadOnlyTimeout, "readOnlyTimeout should be equal") &&
		assert.Equal(t, act.ReadOnlyMaxFailures, exp.ReadOnlyMaxFailures, "readOnlyMaxFailures should be equal")
}

func assertPlugin(t *testing.T, act, exp *Plugin, msg ...interface{}) bool {
	return assertCircuitBreaks(t, act.CircuitBreaks, exp.CircuitBreaks, "circuitBreaks should be equal") && assertConcurrencyControls(t, act.ConcurrencyControls, exp.ConcurrencyControls, "concurrencyControls should be equal")
}

func assertCircuitBreaks(t *testing.T, act, exp []CircuitBreak, msg ...interface{}) bool {
	return assert.ElementsMatch(t, act, exp, "circuitBreaks should be equal")
}

func assertConcurrencyControls(t *testing.T, act, exp []ConcurrencyControl, msg ...interface{}) bool {
	return assert.ElementsMatch(t, act, exp, "concurrencyControls should be equal")
}
func Test_ReadWriteSplittingDynamicConversion(t *testing.T) {
	builder := &ProxyBuilder{
		VirtualDatabaseService: client.VirtualDatabaseService{
			DatabaseService: client.DatabaseService{
				DatabaseMySQL: &client.DatabaseMySQL{
					Host:          "127.0.0.1",
					Port:          3306,
					DB:            "socksdb",
					User:          "root",
					Password:      "fake_password",
					ServerVersion: "5.7.37",
					PoolSize:      3,
				},
			},
			Name:            "catalogue",
			TrafficStrategy: "catalogue",
		},
		TrafficStrategy: client.TrafficStrategy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "catalogue",
				Namespace: "demotest",
			},
			Spec: client.TrafficStrategySpec{
				LoadBalance: &client.LoadBalance{
					ReadWriteSplitting: &client.ReadWriteSplitting{
						Dynamic: &client.ReadWriteSplittingDynamic{
							DefaultTarget: "",
							Rules: []client.ReadWriteSplittingRule{
								{
									Name:          "write-rule",
									Regex:         []string{"^insert"},
									Target:        "readwrite",
									Type:          "regex",
									AlgorithmName: "roundrobin",
								},
								{
									Name:          "read-rule",
									Regex:         []string{"^select"},
									Target:        "read",
									Type:          "regex",
									AlgorithmName: "roundrobin",
								},
							},
							Discovery: client.ReadWriteDiscovery{
								MasterHighAvailability: &client.MasterHighAvailability{
									User:            "monitor",
									Password:        "monitor",
									MonitorInterval: 1000,
									ConnectionProbe: &client.ConnectionProbe{
										Probe: &client.Probe{
											PeriodMilliseconds:  2000,
											FailureThreshold:    3,
											TimeoutMilliseconds: 200,
										},
									},
									PingProbe: &client.PingProbe{
										Probe: &client.Probe{
											PeriodMilliseconds:  1000,
											TimeoutMilliseconds: 100,
											FailureThreshold:    3,
										},
									},
									ReplicationLagProbe: &client.ReplicationLagProbe{
										Probe: &client.Probe{
											PeriodMilliseconds:  1000,
											TimeoutMilliseconds: 3,
											FailureThreshold:    3,
										},
										MaxReplicationLag: 3,
									},
									ReadOnlyProbe: &client.ReadOnlyProbe{
										Probe: &client.Probe{
											PeriodMilliseconds:  1000,
											TimeoutMilliseconds: 3,
											FailureThreshold:    3,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		DatabaseEndpoints: []client.DatabaseEndpoint{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "catalogue",
					Namespace: "demotest",
					Labels: map[string]string{
						"source": "catalogue",
					},
				},
				Spec: client.DatabaseEndpointSpec{
					Database: client.Database{
						MySQL: &client.MySQL{
							DB:       "socksdb",
							Host:     "catalogue-db.demotest",
							Password: "fake_password",
							Port:     3306,
							User:     "root",
						},
					},
				},
			},
		},
	}

	proxy := builder.Build()
	data, err := json.Marshal(proxy)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%s\n", string(data))
}
