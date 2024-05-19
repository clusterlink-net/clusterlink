// Copyright (c) The ClusterLink Authors.
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

// Copyright (c) 2022 The ClusterLink Authors.
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

// Copyright (C) The ClusterLink Authors.
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

package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/clusterlink-net/clusterlink/pkg/util/jsonapi"
)

// Config specifies a client configuration.
type Config struct {
	// Client is the underlying HTTP client.
	Client *jsonapi.Client
	// BasePath is the server HTTP path for manipulating a specific type of objects.
	BasePath string
	// SampleObject is an instance representing the type returned when getting an object.
	SampleObject any
	// SampleList is an instance representing the type returned when listing objects.
	SampleList any
}

// Client for issuing REST-JSON requests for a specific type of objects.
type Client struct {
	client     *jsonapi.Client
	basePath   string
	objectType reflect.Type
	listType   reflect.Type
}

// Create an object.
func (c *Client) Create(object any) error {
	encoded, err := json.Marshal(object)
	if err != nil {
		return fmt.Errorf("unable to encode object: %w", err)
	}

	resp, err := c.client.Post(c.basePath, encoded)
	if err != nil {
		return fmt.Errorf("unable to create object: %w", err)
	}

	if resp.Status != http.StatusCreated {
		return fmt.Errorf("unable to create object (%d), server returned: %s",
			resp.Status, resp.Body)
	}

	return nil
}

// Update an object.
func (c *Client) Update(object any) error {
	encoded, err := json.Marshal(object)
	if err != nil {
		return fmt.Errorf("unable to encode object: %w", err)
	}

	resp, err := c.client.Put(c.basePath, encoded)
	if err != nil {
		return fmt.Errorf("unable to update object: %w", err)
	}

	if resp.Status != http.StatusNoContent {
		return fmt.Errorf("unable to update object (%d), server returned: %s",
			resp.Status, resp.Body)
	}

	return nil
}

// Get an object.
func (c *Client) Get(name string) (any, error) {
	resp, err := c.client.Get(c.basePath + "/" + name)
	if err != nil {
		return nil, fmt.Errorf("unable to get object: %w", err)
	}

	if resp.Status != http.StatusOK {
		return nil, fmt.Errorf("unable to get object (%d), server returned: %s",
			resp.Status, resp.Body)
	}

	decoded := reflect.New(c.objectType).Interface()
	if err := json.Unmarshal(resp.Body, decoded); err != nil {
		return nil, fmt.Errorf("unable to decode object %v: %w", decoded, err)
	}

	return decoded, nil
}

// Delete an object, either by name or the object itself.
func (c *Client) Delete(object any) error {
	var body []byte
	path := c.basePath

	if name, ok := object.(string); ok {
		// delete by name
		path = path + "/" + name
	} else {
		// delete by object
		encoded, err := json.Marshal(object)
		if err != nil {
			return fmt.Errorf("cannot encode object: %w", err)
		}
		body = encoded
	}

	resp, err := c.client.Delete(path, body)
	if err != nil {
		return fmt.Errorf("unable to delete object: %w", err)
	}

	if resp.Status != http.StatusNoContent {
		return fmt.Errorf("unable to delete object (%d), server returned: %s",
			resp.Status, resp.Body)
	}

	return nil
}

// List all objects.
func (c *Client) List() (any, error) {
	resp, err := c.client.Get(c.basePath)
	if err != nil {
		return nil, fmt.Errorf("unable to list objects: %w", err)
	}

	if resp.Status != http.StatusOK {
		return nil, fmt.Errorf("unable to list objects (%d), server returned: %s",
			resp.Status, resp.Body)
	}

	decoded := reflect.New(c.listType).Interface()
	if err := json.Unmarshal(resp.Body, decoded); err != nil {
		return nil, fmt.Errorf("unable to decode object list %v: %w", decoded, err)
	}

	return decoded, nil
}

// NewClient returns a new REST-JSON client.
func NewClient(config *Config) *Client {
	return &Client{
		client:     config.Client,
		basePath:   config.BasePath,
		objectType: reflect.TypeOf(config.SampleObject),
		listType:   reflect.TypeOf(config.SampleList),
	}
}
