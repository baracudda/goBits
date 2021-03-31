package sqlBits

// UI defined values like sort order, pager info, and requested fields desired
// can be an attack vector (SQL Injection) if not properly sanitized.
// If your class can protect against such attack vectors, define this interface
// so that a SqlBuilder instance can query your object for sanitization methods.
type ISqlSanitizer interface {
	// Returns the array of defined fields available.
	GetDefinedFields() []string
	// Returns TRUE if the fieldname specified is sortable.
	IsFieldSortable( aFieldName string )
	// Return the default sort definition.
	GetDefaultSort() OrderByList
	// Providing click-able headers in tables to easily sort them by a particular field
	// is a great UI feature. However, in order to prevent SQL injection attacks, we
	// must double-check that a supplied field name to order the query by is something
	// we can sort on; this method makes use of the IsFieldSortable()
	// method to determine if the browser supplied field name is one of our possible
	// headers that can be clicked on for sorting purposes.
	GetSanitizedOrderByList( aList OrderByList ) OrderByList
	// Prune the field list to remove any invalid fields.
	GetSanitizedFieldList( aFieldList []string ) []string
}
