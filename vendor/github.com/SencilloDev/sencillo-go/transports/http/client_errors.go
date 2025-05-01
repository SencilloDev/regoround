// Copyright 2025 Sencillo
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

package http

import (
	"fmt"
)

// AppError holds information about an application error
type ClientError struct {
	Status  int
	Details string
}

// ClientError is an interface to determine whether the error from the handler is a server or client error

// Error fulfills the error interface
func (c *ClientError) Error() string {
	return c.Details
}

// Body formats the application error for the caller
func (c *ClientError) Body() []byte {
	return []byte(fmt.Sprintf(`{"error": "%s"}`, c.Details))
}

func (c ClientError) As(target error) bool {
	_, ok := target.(*ClientError)
	return ok
}

// NewAppError creates a new application error
func NewClientError(err error, status int) error {
	return &ClientError{
		Status:  status,
		Details: err.Error(),
	}
}
