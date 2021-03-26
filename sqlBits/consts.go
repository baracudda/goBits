/*
 * Copyright (C) 2021 Blackmoon Info Tech Services
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package sqlBits

// 5 digit code meaning "successful completion/no error".
const SQLSTATE_SUCCESS string = "00000"
// 5 digit code meaning "no data"; e.g. UPDATE/DELETE failed due
// to record(s) defined by WHERE clause returned no rows at all.
const SQLSTATE_NO_DATA string = "02000"
// 5 digit ANSI SQL code meaning a table referenced in the SQL does not exist.
const SQLSTATE_TABLE_DOES_NOT_EXIST string = "42S02"

// The SQL element meaning ascending order when sorting.
const ORDER_BY_ASCENDING string = "ASC"
// The SQL element meaning descending order when sorting.
const ORDER_BY_DESCENDING string = "DESC"
// Sometimes we have a nested query in field list. So in order for
// sql.Builder::getQueryTotals() to work automatically, we need to
// supply a comment hint to start the field list.
const FIELD_LIST_HINT_START string = `/* FIELDLIST */`
// Sometimes we have a nested query in field list. So in order for
// sql.Builder::getQueryTotals() to work automatically, we need to
// supply a comment hint to end the field list.
const FIELD_LIST_HINT_END string = `/* /FIELDLIST */`
// Standard SQL specifies '<>' as NOT EQUAL.
const OPERATOR_NOT_EQUAL string = "<>"
