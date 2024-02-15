// Copyright 2023 The ClusterLink Authors.
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

/**********************************************************/
/* Package netutils contain helper functions for network
/* connection
/**********************************************************/

package net

import (
	"net"
	"regexp"
)

var (
	dnsPattern = `^[a-zA-Z0-9-]{1,63}(\.[a-zA-Z0-9-]{1,63})*$`
	dnsRegex   = regexp.MustCompile(dnsPattern)
)

// IsIP returns true if the input is valid IPv4 or IPv6.
func IsIP(str string) bool {
	return net.ParseIP(str) != nil
}

// IsDNS returns true if the input is valid DNS.
func IsDNS(s string) bool {
	return dnsRegex.MatchString(s)
}
