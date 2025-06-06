// Copyright The HTNN Authors.
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

package networkrbac

import (
	"testing"

	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/matching/common_inputs/network/v3"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name  string
		input string
		err   string
	}{
		{
			name: "validate custom match",
			input: `
{
  "statPrefix": "network_rbac",
  "matcher": {
    "matcherTree": {
      "input": {
        "name": "envoy.matching.inputs.source_ip",
        "typedConfig": {
          "@type": "type.googleapis.com/envoy.extensions.matching.common_inputs.network.v3.SourceIPInput"
        }
      },
      "customMatch": {
        "name": "ip-matcher",
        "typedConfig": {
          "@type": "type.googleapis.com/xds.type.matcher.v3.IPMatcher",
          "rangeMathers": [
          ]
        }
      }
    }
  }
}
			`,
			err: "unknown field \"rangeMathers\"",
		},
		{
			name: "validate custom match url",
			input: `
{
  "statPrefix": "network_rbac",
  "matcher": {
    "matcherTree": {
      "input": {
        "name": "envoy.matching.inputs.source_ip",
        "typedConfig": {
          "@type": "type.googleapis.com/envoy.extensions.matching.common_inputs.network.v3.SourceIPInput"
        }
      },
      "customMatch": {
        "name": "ip-matcher",
        "typedConfig": {
          "@type": "type.googleapis.com/xds.type.matcher.v3.CelMatcher"
        }
      }
    }
  }
}
			`,
			err: "must be type.googleapis.com/xds.type.matcher.v3.IPMatcher",
		},
		{
			name: "validate action",
			input: `
{
  "statPrefix": "network_rbac",
  "matcher": {
    "matcherTree": {
      "input": {
        "name": "envoy.matching.inputs.source_ip",
        "typedConfig": {
          "@type": "type.googleapis.com/envoy.extensions.matching.common_inputs.network.v3.SourceIPInput"
        }
      },
      "customMatch": {
        "name": "ip-matcher",
        "typedConfig": {
          "@type": "type.googleapis.com/xds.type.matcher.v3.IPMatcher",
          "rangeMatchers": [
            {
              "ranges": [
                {
                  "addressPrefix": "127.0.0.1",
                  "prefixLen": 32
                }
              ],
              "onMatch": {
                "action": {
                  "name": "envoy.filters.rbac.action",
                  "typedConfig": {
                    "@type": "type.googleapis.com/envoy.config.rbac.v3.Action",
                    "action": "DENY"
                  }
                }
              }
            }
          ]
        }
      }
    }
  }
}
			`,
			err: "invalid Action.Name: value length must be at least 1 runes",
		},
		{
			name: "validate exact match map",
			input: `
{
  "statPrefix": "network_rbac",
  "matcher": {
    "matcherTree": {
      "input": {
        "name": "envoy.matching.inputs.source_ip",
        "typedConfig": {
          "@type": "type.googleapis.com/envoy.extensions.matching.common_inputs.network.v3.SourceIPInput"
        }
      },
      "exactMatchMap": {
        "map": {
          "rule1": {
            "action": {
              "name": "envoy.filters.rbac.action",
              "typedConfig": {
                "@type": "type.googleapis.com/envoy.config.rbac.v3.Action"
              }
            }
          }
        }
      }
    }
  }
}
			`,
			err: "action configuration is empty for rule rule1",
		},
		{
			name: "validate exact match map with multiple rules",
			input: `
{
  "statPrefix": "network_rbac",
  "matcher": {
    "matcherTree": {
      "input": {
        "name": "envoy.matching.inputs.source_ip",
        "typedConfig": {
          "@type": "type.googleapis.com/envoy.extensions.matching.common_inputs.network.v3.SourceIPInput"
        }
      },
      "exactMatchMap": {
        "map": {
          "rule1": {
            "action": {
              "name": "envoy.filters.rbac.action",
              "typedConfig": {
                "@type": "type.googleapis.com/envoy.config.rbac.v3.Action"
              }
            }
          },
          "rule2": {
            "action": {
              "name": "envoy.filters.rbac.action",
              "typedConfig": {
                "@type": "type.googleapis.com/envoy.config.rbac.v3.Action"
              }
            }
          }
        }
      }
    }
  }
}
			`,
			err: "action configuration is empty for rule",
		},
		{
			name: "validate exact match map with invalid action type",
			input: `
{
  "statPrefix": "network_rbac",
  "matcher": {
    "matcherTree": {
      "input": {
        "name": "envoy.matching.inputs.source_ip",
        "typedConfig": {
          "@type": "type.googleapis.com/envoy.extensions.matching.common_inputs.network.v3.SourceIPInput"
        }
      },
      "exactMatchMap": {
        "map": {
          "rule1": {
            "action": {
              "name": "envoy.filters.rbac.action",
              "typedConfig": {
                "@type": "type.googleapis.com/envoy.config.rbac.v3.Action",
                "name": ""
              }
            }
          }
        }
      }
    }
  }
}
			`,
			err: "action configuration is empty for rule rule1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &CustomConfig{}
			err := protojson.Unmarshal([]byte(tt.input), conf)
			if err == nil {
				err = conf.Validate()
			}
			if tt.err == "" {
				assert.Nil(t, err)
			} else {
				assert.ErrorContains(t, err, tt.err)
			}
		})
	}
}
