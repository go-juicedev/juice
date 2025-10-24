/*
Copyright 2024 eatmoreapple

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package driver

import (
	"database/sql"
	"time"
)

// connectOption is a configuration of the connection.
type connectOption struct {
	MaxIdleConnNum      int
	MaxOpenConnNum      int
	MaxConnLifetime     time.Duration
	MaxIdleConnLifetime time.Duration
}

// ConnectOptionFunc is a function to set the connection option.
type ConnectOptionFunc func(*connectOption)

// ConnectWithMaxIdleConnNum sets the maximum number of idle connections.
func ConnectWithMaxIdleConnNum(num int) ConnectOptionFunc {
	return func(option *connectOption) {
		option.MaxIdleConnNum = num
	}
}

// ConnectWithMaxOpenConnNum sets the maximum number of open connections.
func ConnectWithMaxOpenConnNum(num int) ConnectOptionFunc {
	return func(option *connectOption) {
		option.MaxOpenConnNum = num
	}
}

// ConnectWithMaxConnLifetime sets the maximum lifetime of a connection.
func ConnectWithMaxConnLifetime(d time.Duration) ConnectOptionFunc {
	return func(option *connectOption) {
		option.MaxConnLifetime = d
	}
}

// ConnectWithMaxIdleConnLifetime sets the maximum lifetime of an idle connection.
func ConnectWithMaxIdleConnLifetime(d time.Duration) ConnectOptionFunc {
	return func(option *connectOption) {
		option.MaxIdleConnLifetime = d
	}
}

// Connect connects to the database.
func Connect(driver string, datasource string, opts ...ConnectOptionFunc) (*sql.DB, error) {
	var option connectOption
	for _, opt := range opts {
		opt(&option)
	}
	db, err := sql.Open(driver, datasource)
	if err != nil {
		return nil, err
	}
	if option.MaxIdleConnNum > 0 {
		db.SetMaxIdleConns(option.MaxIdleConnNum)
	}
	if option.MaxOpenConnNum > 0 {
		db.SetMaxOpenConns(option.MaxOpenConnNum)
	}
	if option.MaxConnLifetime > 0 {
		db.SetConnMaxLifetime(option.MaxConnLifetime)
	}
	if option.MaxIdleConnLifetime > 0 {
		db.SetConnMaxIdleTime(option.MaxIdleConnLifetime)
	}
	return db, nil
}
