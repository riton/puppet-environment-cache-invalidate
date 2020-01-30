// Copyright (c) Remi Ferrand
//
// Author(s): Remi Ferrand <remi.ferrand_at_cc.in2p3.fr>, 2020
//
// This software is governed by the CeCILL-B license under French law and
// abiding by the rules of distribution of free software.  You can  use,
// modify and/ or redistribute the software under the terms of the CeCILL-B
// license as circulated by CEA, CNRS and INRIA at the following URL
// "http://www.cecill.info".
//
// As a counterpart to the access to the source code and  rights to copy,
// modify and redistribute granted by the license, users are provided only
// with a limited warranty  and the software's author,  the holder of the
// economic rights,  and the successive licensors  have only  limited
// liability.
//
// In this respect, the user's attention is drawn to the risks associated
// with loading,  using,  modifying and/or developing or reproducing the
// software by the user in light of its specific status of free software,
// that may mean  that it is complicated to manipulate,  and  that  also
// therefore means  that it is reserved for developers  and  experienced
// professionals having in-depth computer knowledge. Users are therefore
// encouraged to load and test the software's suitability as regards their
// requirements in conditions enabling the security of their systems and/or
// data to be ensured and,  more generally, to use and operate it in the
// same conditions as regards security.
//
// The fact that you are presently reading this means that you have had
// knowledge of the CeCILL-B license and that you accept its terms.

package puppetapi

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

const (
	httpTimeout                        = 10 * time.Second
	puppetServerPort                   = 8140
	invalidateEnvironmentCacheEndpoint = "/puppet-admin-api/v1/environment-cache"
)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type PuppetAPI struct {
	Server     string
	httpClient httpDoer
}

func NewPuppetAPI(server, cert, caBundle, pk string) (PuppetAPI, error) {

	httpClient, err := NewTLSAuthenticatedHTTPClient(cert, caBundle, pk)
	if err != nil {
		return PuppetAPI{}, err
	}

	return PuppetAPI{
		Server:     server,
		httpClient: httpClient,
	}, nil
}

func NewPuppetAPIWithHTTPClient(server string, httpClient httpDoer) PuppetAPI {
	return PuppetAPI{
		Server:     server,
		httpClient: httpClient,
	}
}

// curl -v --cert /etc/puppet/ssl/certs/ccpuppet04.in2p3.fr.pem --cacert /etc/puppet/ssl/certs/ca.pem --key /etc/puppet/ssl/private_keys/ccpuppet04.in2p3.fr.pem
// -XDELETE https://ccpuppet04.in2p3.fr:8140/puppet-admin-api/v1/environment-cache
func (a PuppetAPI) InvalidateEnvironmentCache(environment string) error {
	fmt.Printf("[%s] %s\n", a.Server, environment)

	endpoint := fmt.Sprintf("https://%s:%d%s", a.Server, puppetServerPort, invalidateEnvironmentCacheEndpoint)
	req, err := http.NewRequest("DELETE", endpoint, nil)
	if err != nil {
		return errors.Wrap(err, "building HTTP request")
	}

	if environment != "" {
		q := req.URL.Query()
		q.Add("environment", environment)
		req.URL.RawQuery = q.Encode()
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "performing HTTP request")
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("bad HTTP status response. Got %d", resp.StatusCode)
	}

	return nil
}
