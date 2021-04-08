package sqlBits

// Aggregate field names as keys mapped to values on SQL used to calc it.
type Aggregate map[string]string

// Aggregater Obtain an aggregate definition to use.
type Aggregater interface {
	GetAggregateDefinition() Aggregate
}

func (a Aggregate) GetAggregateDefinition() Aggregate {
	return a
}

// RowCountAggregate Simple example used to count total rows returned by a SQL statement.
type RowCountAggregate struct {
	def Aggregate
	Rowcount int64
}
var TotalRowCount = RowCountAggregate{
	def: Aggregate{"rowcount": "count(*)"},
}
func (a RowCountAggregate) GetAggregateDefinition() Aggregate {
	return a.def
}

// CloneAsAggregate Sometimes we want to aggregate the query somehow rather than return data from it.
func (sqlbldr *Builder) CloneAsAggregate( aSqlAggragates Aggregater ) *Builder {
	if aSqlAggragates == nil {
		aSqlAggragates = &TotalRowCount
	}
	theAggregateDef := aSqlAggragates.GetAggregateDefinition()
	theFieldList := make([]string, len(theAggregateDef))
	i := 0
	for k, v := range theAggregateDef {
		theFieldList[i] = v + " AS " + k
		i += 1
	}
	theNewBuilder := *sqlbldr
	return theNewBuilder.ReplaceSelectFieldsWith(&theFieldList)
}
