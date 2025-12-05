/*
Copyright 2025 eatmoreapple

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

package sql

import "database/sql"

// BatchResult is a custom implementation of sql.Result that aggregates
// results from multiple batch operations. It provides methods to accumulate
// results from individual batch executions and maintains cumulative statistics.
//
// This implementation ensures that RowsAffected() returns the cumulative
// count of all affected rows across batches, while LastInsertId() returns
// the ID from the last successful insert operation.
type BatchResult struct {
	totalRowsAffected int64
	lastInsertId      int64
}

// AccumulateResult processes a sql.Result from a batch operation and updates
// the internal statistics. It extracts RowsAffected and LastInsertId from
// the provided result and adds them to the cumulative totals.
//
// Parameters:
//   - result: The sql.Result from a single batch operation
//
// This method safely handles errors from RowsAffected() and LastInsertId()
// calls, only updating values when they can be successfully retrieved.
func (r *BatchResult) AccumulateResult(result sql.Result) {
	if result == nil {
		return
	}

	// Accumulate rows affected from this batch
	if rows, err := result.RowsAffected(); err == nil {
		r.totalRowsAffected += rows
	}

	// Update last insert ID from this batch
	if id, err := result.LastInsertId(); err == nil {
		r.lastInsertId = id
	}
}

// LastInsertId returns the insert ID from the last successful batch operation.
// For batch insert operations, this represents the ID of the last inserted record,
// which is consistent with the behavior expected by middleware components.
func (r *BatchResult) LastInsertId() (int64, error) {
	return r.lastInsertId, nil
}

// RowsAffected returns the total number of rows affected across all batch operations.
// This provides an accurate count of the cumulative impact of the entire batch
// processing, rather than just the count from the final batch.
func (r *BatchResult) RowsAffected() (int64, error) {
	return r.totalRowsAffected, nil
}
