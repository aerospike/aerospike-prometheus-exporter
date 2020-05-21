/*
 * Copyright 2012-2019 Aerospike, Inc.
 *
 * Portions may be licensed to Aerospike, Inc. under one or more contributor
 * license agreements WHICH ARE COMPATIBLE WITH THE APACHE LICENSE, VERSION 2.0.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not
 * use this file except in compliance with the License. You may obtain a copy of
 * the License at http: *www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
 * License for the specific language governing permissions and limitations under
 * the License.
 */
package aerospike

// CDTContext defines Nested CDT context. Identifies the location of nested list/map to apply the operation.
// for the current level.
// An array of CTX identifies location of the list/map on multiple
// levels on nesting.
type CDTContext struct {
	id    int
	value Value
}

// CtxListIndex defines Lookup list by index offset.
// If the index is negative, the resolved index starts backwards from end of list.
// If an index is out of bounds, a parameter error will be returned.
// Examples:
// 0: First item.
// 4: Fifth item.
// -1: Last item.
// -3: Third to last item.
func CtxListIndex(index int) *CDTContext {
	return &CDTContext{0x10, IntegerValue(index)}
}

// CtxListRank defines Lookup list by rank.
// 0 = smallest value
// N = Nth smallest value
// -1 = largest value
func CtxListRank(rank int) *CDTContext {
	return &CDTContext{0x11, IntegerValue(rank)}
}

// CtxListValue defines Lookup list by value.
func CtxListValue(key Value) *CDTContext {
	return &CDTContext{0x13, key}
}

// CtxMapIndex defines Lookup map by index offset.
// If the index is negative, the resolved index starts backwards from end of list.
// If an index is out of bounds, a parameter error will be returned.
// Examples:
// 0: First item.
// 4: Fifth item.
// -1: Last item.
// -3: Third to last item.
func CtxMapIndex(index int) *CDTContext {
	return &CDTContext{0x20, IntegerValue(index)}
}

// CtxMapRank defines Lookup map by rank.
// 0 = smallest value
// N = Nth smallest value
// -1 = largest value
func CtxMapRank(rank int) *CDTContext {
	return &CDTContext{0x21, IntegerValue(rank)}
}

// CtxMapKey defines Lookup map by key.
func CtxMapKey(key Value) *CDTContext {
	return &CDTContext{0x22, key}
}

// CtxMapValue defines Lookup map by value.
func CtxMapValue(key Value) *CDTContext {
	return &CDTContext{0x23, key}
}
