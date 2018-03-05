//
// Copyright (c) 2018 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package adapters

import (
	"encoding/json"
	"fmt"
	"net/http"

	b64 "encoding/base64"

	"github.com/automationbroker/bundle-lib/apb"
	log "github.com/sirupsen/logrus"
)

const openShiftName = "registry.connect.redhat.com"
const openShiftAuthURL = "https://sso.redhat.com/auth/realms/rhc4tp/protocol/docker-v2/auth?service=docker-registry"
const openShiftManifestURL = "https://registry.connect.redhat.com/v2/%v/manifests/%v"

// OpenShiftAdapter - Docker Hub Adapter
type OpenShiftAdapter struct {
	Config Configuration
}

// OpenShiftImage - Image from a OpenShift registry.
type OpenShiftImage struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// RegistryName - Retrieve the registry name
func (r OpenShiftAdapter) RegistryName() string {
	return openShiftName
}

// GetImageNames - retrieve the images
func (r OpenShiftAdapter) GetImageNames() ([]string, error) {
	log.Debug("OpenShiftAdapter::GetImageNames")
	log.Debug("BundleSpecLabel: %s", BundleSpecLabel)

	images := r.Config.Images
	log.Debug("Configured to use images: %v", images)

	return images, nil
}

// FetchSpecs - retrieve the spec for the image names.
func (r OpenShiftAdapter) FetchSpecs(imageNames []string) ([]*apb.Spec, error) {
	log.Debug("OpenShiftAdapter::FetchSpecs")
	specs := []*apb.Spec{}
	for _, imageName := range imageNames {
		log.Debug("%v", imageName)
		spec, err := r.loadSpec(imageName)
		if err != nil {
			log.Errorf("Failed to retrieve spec data for image %s - %v", imageName, err)
		}
		if spec != nil {
			specs = append(specs, spec)
		}
	}
	return specs, nil
}

// getOpenShiftToken - will retrieve the docker hub token.
func (r OpenShiftAdapter) getOpenShiftAuthToken() (string, error) {
	type TokenResponse struct {
		Token string `json:"token"`
	}
	username := r.Config.User
	password := r.Config.Pass
	authString := fmt.Sprintf("%v:%v", username, password)

	authString = b64.StdEncoding.EncodeToString([]byte(authString))

	req, err := http.NewRequest("GET", openShiftAuthURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Basic %v", authString))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	tokenResp := TokenResponse{}
	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	if err != nil {
		return "", err
	}
	return tokenResp.Token, nil
}

func (r OpenShiftAdapter) loadSpec(imageName string) (*apb.Spec, error) {
	log.Debug("OpenShiftAdapter::LoadSpec")
	if r.Config.Tag == "" {
		r.Config.Tag = "latest"
	}
	req, err := http.NewRequest("GET", fmt.Sprintf(openShiftManifestURL, imageName, r.Config.Tag), nil)
	if err != nil {
		return nil, err
	}
	token, err := r.getOpenShiftAuthToken()
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	return imageToSpec(req, fmt.Sprintf("%s/%s:%s", r.RegistryName(), imageName, r.Config.Tag))
}
