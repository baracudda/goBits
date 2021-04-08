package sqlBits

// SQLSTATE_SUCCESS 5 digit code meaning "successful completion/no error".
const SQLSTATE_SUCCESS string = "00000"
// SQLSTATE_NO_DATA 5 digit code meaning "no data"; e.g. UPDATE/DELETE failed due
// to record(s) defined by WHERE clause returned no rows at all.
const SQLSTATE_NO_DATA string = "02000"
// SQLSTATE_TABLE_DOES_NOT_EXIST 5 digit ANSI SQL code meaning a table referenced
// in the SQL does not exist.
const SQLSTATE_TABLE_DOES_NOT_EXIST string = "42S02"

// ORDER_BY_ASCENDING The SQL element meaning ascending order when sorting.
const ORDER_BY_ASCENDING string = "ASC"
// ORDER_BY_DESCENDING The SQL element meaning descending order when sorting.
const ORDER_BY_DESCENDING string = "DESC"
// FIELD_LIST_HINT_START Sometimes we have a nested query in field list.
// So in order for sql.Builder::getQueryTotals() to work automatically,
// we need to supply a comment hint to start the field list.
const FIELD_LIST_HINT_START string = `/* FIELDLIST */`
// FIELD_LIST_HINT_END Sometimes we have a nested query in field list.
// So in order for sql.Builder::getQueryTotals() to work automatically,
// we need to supply a comment hint to end the field list.
const FIELD_LIST_HINT_END string = `/* /FIELDLIST */`
// OPERATOR_NOT_EQUAL Standard SQL specifies '<>' as NOT EQUAL.
const OPERATOR_NOT_EQUAL string = "<>"
