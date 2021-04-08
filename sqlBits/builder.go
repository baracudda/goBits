package sqlBits

import (
	"regexp"
	"strconv"
	"strings"
)

type DbModeler interface {
	DbMetatater
	DbTransactioner
}

type IDataSource interface {
	IsKeyDefined( aKey string ) bool
	IsKeyValueAList( aKey string ) bool
	GetValueForKey( aKey string ) *string
	GetValueListForKey( aKey string ) *[]string
}

// OrderByList Keys are field names, values are either ORDER_BY_* consts: 'ASC' or 'DESC'.
type OrderByList map[string]string

// Builder Use this class to help build SQL queries.
// Supports: MySQL and Postgres.
type Builder struct {
	// Database model used to tweak SQL dialect specifics.
	myDbModel       DbModeler
	// Used to determine if we started a transaction or not.
	// The flag is incremented every time a transaction is requested
	//   and decremented when commited; only begins/commits when transitioning
	//   from 0 to 1 and back to 0. This allows us to "nest" transactions.
	myTransactionFlag int
	// If set, parameter data is retrieved from it.
	myDataSource IDataSource

	/**
	 * The object used to sanitize field/orderby lists to help prevent
	 * SQL injection attacks.
	 * @var ISqlSanitizer
	 */
	//public $mySqlSanitizer = null;

	// The SQL string being built.
	mySql           string
	// SQL statement parameters to use (contains all keys from mySetParams, too).
	myParams        map[string]*string
	// used by SQL() if driver does not support named parameters
	myOrdQuerySql   string
	// used by SQL() if driver does not support named parameters
	myOrdQueryArgs  []interface{}
	// SQL statement set parameters to use.
	mySetParams     map[string]*[]string
	// SQL statement parameter types (not sure if we need them, yet)
	//myParamTypes    map[string]string
	// Prefix for a parameter about to be added.
	myParamPrefix   string
	// Operator for the parameter to use. e.g. " LIKE ", "=", "<>", etc.
	myParamOperator string

	// Using the "=" when NULL is involved is ambiguous unless you know
	// if it is part of a SET clause or WHERE clause.  Explicitly set
	// this flag to let the SqlBuilder know it is in a WHERE clause.
	bUseIsNull bool
}

// NewBuilder Models can use this package to help build their SQL queries.
func NewBuilder( aDbModeler DbModeler ) *Builder {
	return new(Builder).WithModel(aDbModeler)
}

// WithModel Initializer like NewBuilder. e.g.: new(Builder).WithModel(aDbModeler)
func (sqlbldr *Builder) WithModel( aDbModeler DbModeler ) *Builder {
	if aDbModeler == nil {
		panic("no DbModeler defined!")
	}
	sqlbldr.myDbModel = aDbModeler
	return sqlbldr.Reset()
}

// Reset Resets the object so it can be resused without creating a new instance.
func (sqlbldr *Builder) Reset() *Builder {
	sqlbldr.mySql = ""
	sqlbldr.myParams = map[string]*string{}
	sqlbldr.mySetParams = map[string]*[]string{}
	//sqlbldr.myParamTypes = map[string]string{}
	sqlbldr.myParamPrefix = " "
	sqlbldr.myParamOperator = "="
	sqlbldr.bUseIsNull = false
	return sqlbldr
}

// BeginTransaction If we are not already in a transaction, start one.
func (sqlbldr *Builder) BeginTransaction() *Builder {
	if sqlbldr.myTransactionFlag < 1 {
		if !sqlbldr.myDbModel.InTransaction() {
			sqlbldr.myDbModel.BeginTransaction()
		}
	}
	sqlbldr.myTransactionFlag += 1
	return sqlbldr
}

// CommitTransaction If we started a transaction earlier, commit it.
func (sqlbldr *Builder) CommitTransaction() *Builder {
	if sqlbldr.myTransactionFlag > 0 {
		if sqlbldr.myTransactionFlag -= 1; sqlbldr.myTransactionFlag == 0 {
			sqlbldr.myDbModel.CommitTransaction()
		}
	}
	return sqlbldr
}

// RollbackTransaction If we started a transaction earlier, roll it back.
func (sqlbldr *Builder) RollbackTransaction() *Builder {
	if sqlbldr.myTransactionFlag > 0 {
		if sqlbldr.myTransactionFlag -= 1; sqlbldr.myTransactionFlag == 0 {
			sqlbldr.myDbModel.RollbackTransaction()
		}
	}
	return sqlbldr
}

// GetQuoted Quoted identifiers are DB vendor specific so providing a helper method
// to just return a properly quoted string for MySQL vs MSSQL vs Oracle, etc. is handy.
func (sqlbldr *Builder) GetQuoted( aIdentifier string ) string {
	delim := string(sqlbldr.myDbModel.GetDbMeta().IdentifierDelimiter)
	return delim + strings.Replace(aIdentifier, delim, delim+delim, -1) + delim
}

// StartWith Sets the SQL string to this value to build upon.
func (sqlbldr *Builder) StartWith( aSql string ) *Builder {
	sqlbldr.mySql = aSql
	return sqlbldr
}

// StartFilter Some operators require alternate handling during WHERE clauses
// (e.g. "=" with NULLs). Similar to StartWhereClause(), this method is
// specific to building a filter that consists entirely of a
// partial WHERE clause which will get appended to the main SqlBuilder
// using ApplyFilter().
func (sqlbldr *Builder) StartFilter() *Builder {
	sqlbldr.bUseIsNull = true
	driverName := sqlbldr.myDbModel.GetDbMeta().Name
	switch driverName {
	case MySQL:
		sqlbldr.StartWith("1")
	case PostgreSQL:
		sqlbldr.StartWith("true")
	default:
		sqlbldr.StartWith("true")
	}//switch
	return sqlbldr.SetParamPrefix(" AND ")
}

// SetDataSource Set our param value source.
func (sqlbldr *Builder) SetDataSource( aDataSource IDataSource ) *Builder {
	sqlbldr.myDataSource = aDataSource
	return sqlbldr
}

// SetParam Sets the param value and param type, but does not affect the SQL string.
func (sqlbldr *Builder) SetParam( aParamKey string, aParamValue string ) *Builder {
	s := aParamValue
	return sqlbldr.SetNullableParam(aParamKey, &s)
}

// SetNullableParam Sets the param value and param type, but does not affect the SQL string.
func (sqlbldr *Builder) SetNullableParam( aParamKey string, aParamValue *string ) *Builder {
	sqlbldr.myParams[aParamKey] = aParamValue
	//sqlbldr.myParamTypes[aParamKey] = "string"
	return sqlbldr
}

// SetParamSet Sets the param value set, but does not affect the SQL string.
func (sqlbldr *Builder) SetParamSet( aParamKey string, aParamValues *[]string ) *Builder {
	sqlbldr.myParams[aParamKey] = nil
	sqlbldr.mySetParams[aParamKey] = aParamValues
	//sqlbldr.myParamTypes[aParamKey] = "string"
	return sqlbldr
}

// IsParamASet Inquire if the data that will be used for a particular param is a set or not.
func (sqlbldr *Builder) IsParamASet( aParamKey string ) bool {
	_, ok := sqlbldr.myParams[aParamKey]
	if ok {
		//if key exists in myParams, then result is if key also exists in mySetParams
		_, ok := sqlbldr.mySetParams[aParamKey]
		return ok
	} else if sqlbldr.myDataSource != nil {
		//if key non-existant, check myDataSource
		return sqlbldr.myDataSource.IsKeyValueAList(aParamKey)
	}
	//no clue about this param, return false
	return false
}

// GetParam Gets the current value of a param that has been added.
func (sqlbldr *Builder) GetParam( aParamKey string ) *string {
	val, ok := sqlbldr.myParams[aParamKey]
	if ok {
		return val
	} else {
		return nil
	}
}

// GetParamSet Gets the current value of a param that has been added.
func (sqlbldr *Builder) GetParamSet( aParamKey string ) *[]string {
	valSet, ok := sqlbldr.mySetParams[aParamKey]
	if ok {
		return valSet
	} else {
		return nil
	}
}

// GetUniqueParamKey Some SQL drivers require all query parameters be unique.
// This poses an issue when multiple datakeys with the same name are needed in
// the query (especially true for MERGE queries). This method will check for any
// existing parameters named aParamKey and will return a new name with a number
// for a suffix to ensure its uniqueness.
func (sqlbldr *Builder) GetUniqueParamKey( aParamKey string ) string {
	i := 1
	theKey := aParamKey
	_, bKeyExists := sqlbldr.myParams[theKey]
	for bKeyExists {
		i += 1
		theKey = aParamKey + strconv.Itoa(i)
		_, bKeyExists = sqlbldr.myParams[theKey]
	}
	return theKey
}

// StartWhereClause Some operators require alternate handling during WHERE
// clauses (e.g. "=" with NULLs). This will setParamPrefix(" WHERE ") which will
// apply to the next AddParam.
func (sqlbldr *Builder) StartWhereClause() *Builder {
	sqlbldr.bUseIsNull = true
	return sqlbldr.SetParamPrefix(" WHERE ")
}

// EndWhereClause Resets the WHERE clause flag.
func (sqlbldr *Builder) EndWhereClause() *Builder {
	sqlbldr.bUseIsNull = false
	return sqlbldr
}

// Add Adds a string to the SQL prefixed with a space (just in case).
// *DO NOT* use this method to write values gathered from
// user input directly into a query. *ALWAYS* use the
// .AddParam() or similar methods, or pre-sanitize the data
// value before writing it into the query.
func (sqlbldr *Builder) Add( aStr string ) *Builder {
	sqlbldr.mySql += " " + aStr
	return sqlbldr
}

// SetParamPrefix Sets the "glue" string that gets prepended to all subsequent calls to
// AddParam kinds of methods. Spacing is important here, so add what is needed!
func (sqlbldr *Builder) SetParamPrefix( aStr string ) *Builder {
	sqlbldr.myParamPrefix = aStr
	return sqlbldr
}

// SetParamOperator Operator string to use in all subsequent calls to addParam
// methods. "=" is default, " LIKE " is a popular operator as well.
func (sqlbldr *Builder) SetParamOperator( aStr string ) *Builder {
	// "!=" is not standard SQL, but is a common programmer mistake, cnv to "<>"
	aStr = strings.Replace(aStr, "!=", OPERATOR_NOT_EQUAL, -1)
	sqlbldr.myParamOperator = aStr
	return sqlbldr
}

// getParamValueFromDataSource Retrieve the data that will be used for a particular param.
func (sqlbldr *Builder) getParamValueFromDataSource( aParamKey string ) *Builder {
	if sqlbldr.myDataSource != nil {
		if sqlbldr.myDataSource.IsKeyValueAList(aParamKey) {
			return sqlbldr.SetParamSet(aParamKey, sqlbldr.myDataSource.GetValueListForKey(aParamKey))
		} else {
			return sqlbldr.SetNullableParam(aParamKey, sqlbldr.myDataSource.GetValueForKey(aParamKey))
		}
	}
	return sqlbldr
}

// SetParamValueIfNull Set a value for a param when its data value is NULL.
func (sqlbldr *Builder) SetParamValueIfNull( aParamKey string, aNewValue string ) *Builder {
	sqlbldr.getParamValueFromDataSource(aParamKey)
	isSet := sqlbldr.IsParamASet(aParamKey)
	if valSet := sqlbldr.GetParamSet(aParamKey); isSet && (valSet == nil || len(*valSet) == 0) {
		sqlbldr.SetParam(aParamKey, aNewValue)
	} else if !isSet && sqlbldr.GetParam(aParamKey) == nil {
		sqlbldr.SetParam(aParamKey, aNewValue)
	}
	return sqlbldr
}

// SetParamValueIfEmpty Set a value for a param when its data value is empty(). e.g. null|""|0
func (sqlbldr *Builder) SetParamValueIfEmpty( aParamKey string, aNewValue string ) *Builder {
	sqlbldr.getParamValueFromDataSource(aParamKey)
	isSet := sqlbldr.IsParamASet(aParamKey)
	if valSet := sqlbldr.GetParamSet(aParamKey); isSet && (valSet == nil || len(*valSet) == 0) {
		sqlbldr.SetParam(aParamKey, aNewValue)
	} else if val := sqlbldr.GetParam(aParamKey) ; !isSet && (val == nil || *val == "" || *val == "0") {
		sqlbldr.SetParam(aParamKey, aNewValue)
	}
	return sqlbldr
}

// isDataKeyDefined Mainly used internally by AddParamIfDefined to determine if data param exists.
func (sqlbldr *Builder) isDataKeyDefined( aDataKey string ) bool {
	if sqlbldr.myDataSource != nil {
		return sqlbldr.myDataSource.IsKeyDefined(aDataKey)
	} else {
		return false
	}
}

// addParamAsListForColumn Adds to the SQL string as a set of values;
// e.g. "(:paramkey_1,:paramkey_2,:paramkey_N)"
// Honors the ParamPrefix and ParamOperator properties.
func (sqlbldr *Builder) addParamAsListForColumn( aColumnName string,
	aParamKey string, aDataValuesList *[]string,
) *Builder {
	if aDataValuesList != nil && len(*aDataValuesList) > 0 {
		sqlbldr.mySql += sqlbldr.myParamPrefix + sqlbldr.GetQuoted(aColumnName)
		sqlbldr.mySql += sqlbldr.myParamOperator + "("
		i := 1
		for _, val := range *aDataValuesList {
			theParamKey := aParamKey + "_" + strconv.Itoa(i)
			i += 1
			sqlbldr.mySql += ":" + theParamKey + ","
			sqlbldr.SetParam(theParamKey, val)
		}
		sqlbldr.mySql = strings.TrimRight(sqlbldr.mySql, ",") + ")"
	}
	return sqlbldr
}

// addingParam Internal method to affect SQL statment with a param and its value.
func (sqlbldr *Builder) addingParam( aColName string, aParamKey string ) {
	isSet := sqlbldr.IsParamASet(aParamKey)
	if valSet := sqlbldr.GetParamSet(aParamKey); isSet && valSet != nil && len(*valSet) > 0 {
		saveParamOp := sqlbldr.myParamOperator
		switch strings.TrimSpace(sqlbldr.myParamOperator) {
		case "=":
			sqlbldr.myParamOperator = " IN "
		case OPERATOR_NOT_EQUAL:
			sqlbldr.myParamOperator = " NOT IN "
		}//switch
		sqlbldr.addParamAsListForColumn(aColName, aParamKey, valSet)
		sqlbldr.myParamOperator = saveParamOp
	} else {
		if val := sqlbldr.GetParam(aParamKey); val != nil || !sqlbldr.bUseIsNull {
			sqlbldr.mySql += sqlbldr.myParamPrefix + sqlbldr.GetQuoted(aColName) +
				sqlbldr.myParamOperator + ":" + aParamKey
		} else {
			switch strings.TrimSpace(sqlbldr.myParamOperator) {
			case "=":
				sqlbldr.mySql += sqlbldr.myParamPrefix + sqlbldr.GetQuoted(aColName) + " IS NULL"
			case OPERATOR_NOT_EQUAL:
				sqlbldr.mySql += sqlbldr.myParamPrefix + sqlbldr.GetQuoted(aColName) + " IS NOT NULL"
			}//switch
		}
	}
}

// AppendParam Parameter must go into the SQL string regardless of NULL status of data.
func (sqlbldr *Builder) AppendParam( aParamKey string, aParamValue string ) *Builder {
	sqlbldr.SetParam(aParamKey, aParamValue)
	sqlbldr.addingParam(aParamKey, aParamKey)
	return sqlbldr
}

// MustAddParam Parameter must go into the SQL string regardless of NULL status of data.
func (sqlbldr *Builder) MustAddParam( aParamKey string ) *Builder {
	return sqlbldr.MustAddParamForColumn(aParamKey, aParamKey)
}

// MustAddParamForColumn Parameter must go into the SQL string regardless of NULL
// status of data. This is a "shortcut" designed to combine calls to setParamValue, and addParam.
func (sqlbldr *Builder) MustAddParamForColumn( aParamKey string, aColumnName string ) *Builder {
	sqlbldr.getParamValueFromDataSource(aParamKey)
	sqlbldr.addingParam(aColumnName, aParamKey)
	return sqlbldr
}

// AddParamIfDefined Parameter only gets added to the SQL string if data IS NOT NULL.
func (sqlbldr *Builder) AddParamIfDefined( aParamKey string ) *Builder {
	if sqlbldr.isDataKeyDefined(aParamKey) {
		sqlbldr.getParamValueFromDataSource(aParamKey)
		sqlbldr.addingParam(aParamKey, aParamKey)
	}
	return sqlbldr
}

// AddParamForColumnIfDefined Parameter only gets added to the SQL string if data IS NOT NULL.
func (sqlbldr *Builder) AddParamForColumnIfDefined( aParamKey string, aColumnName string ) *Builder {
	if sqlbldr.isDataKeyDefined(aParamKey) {
		sqlbldr.getParamValueFromDataSource(aParamKey)
		sqlbldr.addingParam(aColumnName, aParamKey)
	}
	return sqlbldr
}

// AddFieldList Adds the list of fields (columns) to the SQL string.
func (sqlbldr *Builder) AddFieldList( aFieldList *[]string ) *Builder {
	theFieldListStr := sqlbldr.myParamPrefix + "*"
	if aFieldList != nil && len(*aFieldList) > 0 {
		theFieldListStr = sqlbldr.myParamPrefix +
			strings.Join(*aFieldList, ", "+sqlbldr.myParamPrefix)
	}
	return sqlbldr.Add(theFieldListStr)
}

// AddQueryLimit Return the SQL "LIMIT" expression for our model's database type.
func (sqlbldr *Builder) AddQueryLimit( aLimit int, aOffset int ) *Builder {
	if aLimit > 0 && sqlbldr.myDbModel != nil {
		driverName := sqlbldr.myDbModel.GetDbMeta().Name
		switch driverName {
		case MySQL:
		case PostgreSQL:
		default:
			sqlbldr.Add("LIMIT").Add(strconv.Itoa(aLimit))
			if aOffset > 0 {
				sqlbldr.Add("OFFSET").Add(strconv.Itoa(aOffset))
			}
		}//switch
	}
	return sqlbldr
}

// AddSubQueryForColumn Sub-query gets added to the SQL string.
func (sqlbldr *Builder) AddSubQueryForColumn( aSubQuery *Builder, aColumnName string ) *Builder {
	saveParamOp := sqlbldr.myParamOperator
	switch strings.TrimSpace(sqlbldr.myParamOperator) {
	case "=":
		sqlbldr.myParamOperator = " IN "
	case OPERATOR_NOT_EQUAL:
		sqlbldr.myParamOperator = " NOT IN "
	}//switch
	sqlbldr.mySql += sqlbldr.myParamPrefix + sqlbldr.GetQuoted(aColumnName) +
		sqlbldr.myParamOperator + "(" + aSubQuery.mySql + ")"
	sqlbldr.myParamOperator = saveParamOp
	//also merge in any params from the sub-query
	for k, v := range aSubQuery.myParams {
		sqlbldr.myParams[k] = v
	}
	for k, v := range aSubQuery.mySetParams {
		sqlbldr.mySetParams[k] = v
	}
	return sqlbldr
}

// ApplyFilter Apply an externally defined set of WHERE field clauses and param
// values to our SQL (excludes the "WHERE" keyword).
func (sqlbldr *Builder) ApplyFilter( aFilter *Builder ) *Builder {
	if aFilter != nil {
		if aFilter.mySql != "" {
			sqlbldr.mySql += sqlbldr.myParamPrefix + aFilter.mySql
		}
		//also merge in any params from the sub-query
		for k, v := range aFilter.myParams {
			sqlbldr.myParams[k] = v
		}
		for k, v := range aFilter.mySetParams {
			sqlbldr.mySetParams[k] = v
		}
	}
	return sqlbldr
}

// ApplySortList If sort list is defined and its contents are also contained
// in the non-empty $aFieldList, then apply the sort order as neccessary.
// @see ApplyOrderByList() which this method is an alias of.
func (sqlbldr *Builder) ApplySortList( aSortList *OrderByList ) *Builder {
	return sqlbldr.ApplyOrderByList(aSortList)
}

// ApplyOrderByList If order by list is defined, then apply the sort order as neccessary.
func (sqlbldr *Builder) ApplyOrderByList( aOrderByList *OrderByList ) *Builder {
	if aOrderByList != nil && sqlbldr.myDbModel != nil {
		theSortKeyword := "ORDER BY"
		/* in case we find diff keywords later...
		driverName := sqlbldr.myDbModel.GetDbMeta().Name
		switch driverName {
		case MySQL:
		case PostgreSQL:
		default:
			theSortKeyword = "ORDER BY"
		}//switch
		 */
		sqlbldr.Add(theSortKeyword)

		theOrderByList := make([]string, len(*aOrderByList))
		idx := 0
		for k, v := range *aOrderByList {
			theEntry := k + " "
			if strings.ToUpper(strings.TrimSpace(v)) == ORDER_BY_DESCENDING {
				theEntry += ORDER_BY_DESCENDING
			} else {
				theEntry += ORDER_BY_ASCENDING
			}
			theOrderByList[idx] = theEntry
			idx += 1
		}
		sqlbldr.Add(strings.Join(theOrderByList, ","))
	}
	return sqlbldr
}

// ReplaceSelectFieldsWith Replace the currently formed SELECT fields with the param.
// If you have nested queries, you will need to use the FIELD_LIST_HINT_* consts in
// the SQL like so:
// "SELECT /* FIELDLIST */ field1, field2, (SELECT blah) AS field3 /* /FIELDLIST */ FROM"
func (sqlbldr *Builder) ReplaceSelectFieldsWith( aSelectFields *[]string ) *Builder {
	if aSelectFields != nil && len(*aSelectFields) > 0 {
		var re *regexp.Regexp
		//nested queries can mess us up, so check for hints first
		if strings.Index(sqlbldr.mySql, FIELD_LIST_HINT_START) > 0 &&
			strings.Index(sqlbldr.mySql, FIELD_LIST_HINT_END) > 0 {
			re = regexp.MustCompilePOSIX("(?i)SELECT /* FIELDLIST */ .+? /* /FIELDLIST */ FROM")
		} else {
			//we want a "non-greedy" match so that it stops at the first "FROM" it finds: ".+?"
			re = regexp.MustCompilePOSIX("(?i)SELECT .+? FROM")
		}
		sqlbldr.mySql = re.ReplaceAllString(sqlbldr.mySql, strings.Join(*aSelectFields, ", "))
	}
	return sqlbldr
}

// GetSQLStatement Return our currently built SQL statement.
func (sqlbldr *Builder) GetSQLStatement() string {
	return sqlbldr.mySql
}

// SQL Return our currently built SQL statement.
func (sqlbldr *Builder) SQL() string {
	if sqlbldr.myParams != nil && len(sqlbldr.myParams) > 0 &&
		sqlbldr.myDbModel != nil && !sqlbldr.myDbModel.GetDbMeta().SupportsNamedParams {
		sqlbldr.myOrdQuerySql = sqlbldr.mySql
		i := 1
		for k, v := range sqlbldr.myParams {
			theOldKey := ":"+k
			theNewKey := "$"+strconv.Itoa(i)
			if strings.Contains(sqlbldr.myOrdQuerySql, theOldKey) && v != nil {
				sqlbldr.myOrdQuerySql = strings.Replace(sqlbldr.myOrdQuerySql, theOldKey, theNewKey, 1)
				sqlbldr.myOrdQueryArgs = append(sqlbldr.myOrdQueryArgs, *v)
				i += 1
			}
		}
		return sqlbldr.myOrdQuerySql
	} else {
		return sqlbldr.mySql
	}
}

// SQLparams Return our current SQL params in use.
func (sqlbldr *Builder) SQLparams() map[string]*string {
	if sqlbldr.myParams != nil {
		return sqlbldr.myParams
	} else {
		return map[string]*string{}
	}
}

// SQLparamSets Return our current SQL param sets in use.
func (sqlbldr *Builder) SQLparamSets() map[string]*[]string {
	if sqlbldr.mySetParams != nil {
		return sqlbldr.mySetParams
	} else {
		return map[string]*[]string{}
	}
}

// SQLargs Return SQL query arguments IFF the driver does not support named parameters.
func (sqlbldr *Builder) SQLargs() []interface{} {
	return sqlbldr.myOrdQueryArgs
}

// SQLnamedArgs Return SQL query arguments as named parameters.
func (sqlbldr *Builder) SQLnamedArgs() map[string]interface{} {
	theResults := map[string]interface{}{}
	for k, v := range sqlbldr.myParams {
		if v != nil {
			theResults[k] = *v
		}
	}
	return theResults
}
