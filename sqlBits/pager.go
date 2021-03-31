package sqlBits

// UI defined values like sort order, pager info, and requested fields desired
// can be an attack vector (SQL Injection) if not properly sanitized.
// If your class can protect against such attack vectors, define this interface
// so that a SqlBuilder instance can query your object for sanitization methods.
type IPagedResults interface {
	// Pagers and Long processes may desire a total regardless of pager use.
	IsTotalRowCountDesired() bool
	// Get the defined maximum row count for a page in a pager. 0 means no limit.
	GetPagerPageSize() int64
	// Get the SQL query offset based on pager page size and page desired.
	GetPagerQueryOffset() int64
}

type ResultsWithRowCounter interface {
	// Set the query total regardless of paging or not.
	SetTotalRowCount( aTotalRowCount int64 )
}
