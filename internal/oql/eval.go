package oql

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/modbender/hprof-analyzer/internal/index"
)

// Result represents one row in the OQL query result.
type Result struct {
	Values map[string]string
}

// Eval evaluates an OQL query against an index.
func Eval(stmt *SelectStmt, idx *index.Index) ([]string, []Result, error) {
	// Find matching objects from FROM clause
	matchingObjects := findMatchingObjects(stmt.From, idx)

	// Determine if this is an aggregate query
	isAggregate := false
	for _, col := range stmt.Columns {
		if _, ok := col.Expr.(FuncCall); ok {
			fc := col.Expr.(FuncCall)
			if isAggregateFunc(fc.Name) {
				isAggregate = true
				break
			}
		}
	}

	if isAggregate && len(stmt.GroupBy) > 0 {
		return evalGroupBy(stmt, idx, matchingObjects)
	}

	if isAggregate {
		return evalAggregate(stmt, idx, matchingObjects)
	}

	return evalSimple(stmt, idx, matchingObjects)
}

func findMatchingObjects(from FromClause, idx *index.Index) []uint64 {
	var result []uint64
	className := from.ClassName

	for objID, obj := range idx.Objects {
		var objClassName string
		switch obj.Kind {
		case index.KindInstance:
			objClassName = idx.ClassName(obj.ClassID)
		case index.KindClass:
			objClassName = idx.ClassName(obj.ID)
		case index.KindObjArray:
			objClassName = idx.ClassName(obj.ClassID) + "[]"
		case index.KindPrimArray:
			continue // skip primitive arrays for class matching
		}

		if from.Instanceof {
			// Match class and subclasses
			if matchesInstanceof(obj, className, idx) {
				result = append(result, objID)
			}
		} else {
			if objClassName == className {
				result = append(result, objID)
			}
		}
	}

	return result
}

func matchesInstanceof(obj *index.ObjectEntry, targetClass string, idx *index.Index) bool {
	classID := obj.ClassID
	if obj.Kind == index.KindClass {
		classID = obj.ID
	}

	// Walk class hierarchy
	for classID != 0 {
		if idx.ClassName(classID) == targetClass {
			return true
		}
		ce, ok := idx.Classes[classID]
		if !ok {
			break
		}
		classID = ce.SuperClassObjID
	}
	return false
}

func evalSimple(stmt *SelectStmt, idx *index.Index, objects []uint64) ([]string, []Result, error) {
	// Build column headers
	headers := buildHeaders(stmt.Columns)

	var results []Result
	for _, objID := range objects {
		// Apply WHERE filter
		if stmt.Where != nil {
			match, err := evalBoolExpr(stmt.Where, objID, idx)
			if err != nil {
				return nil, nil, err
			}
			if !match {
				continue
			}
		}

		row := make(map[string]string, len(headers))
		for i, col := range stmt.Columns {
			val := evalColumnExpr(col.Expr, objID, idx)
			row[headers[i]] = val
		}
		results = append(results, Result{Values: row})
	}

	// Apply ORDER BY
	if len(stmt.OrderBy) > 0 {
		sortResults(results, stmt.OrderBy)
	}

	// Apply LIMIT
	if stmt.Limit > 0 && len(results) > stmt.Limit {
		results = results[:stmt.Limit]
	}

	return headers, results, nil
}

func evalAggregate(stmt *SelectStmt, idx *index.Index, objects []uint64) ([]string, []Result, error) {
	headers := buildHeaders(stmt.Columns)

	// Filter objects
	var filtered []uint64
	for _, objID := range objects {
		if stmt.Where != nil {
			match, err := evalBoolExpr(stmt.Where, objID, idx)
			if err != nil {
				return nil, nil, err
			}
			if !match {
				continue
			}
		}
		filtered = append(filtered, objID)
	}

	row := make(map[string]string, len(headers))
	for i, col := range stmt.Columns {
		val := evalAggregateExpr(col.Expr, filtered, idx)
		row[headers[i]] = val
	}

	return headers, []Result{{Values: row}}, nil
}

func evalGroupBy(stmt *SelectStmt, idx *index.Index, objects []uint64) ([]string, []Result, error) {
	headers := buildHeaders(stmt.Columns)

	// Group objects
	groups := make(map[string][]uint64)
	for _, objID := range objects {
		if stmt.Where != nil {
			match, err := evalBoolExpr(stmt.Where, objID, idx)
			if err != nil {
				return nil, nil, err
			}
			if !match {
				continue
			}
		}

		// Build group key
		var keyParts []string
		for _, gb := range stmt.GroupBy {
			fa := FieldAccess{Field: gb}
			if strings.HasPrefix(gb, "@") {
				fa.Field = gb[1:]
				fa.IsBuiltin = true
			}
			keyParts = append(keyParts, evalColumnExpr(fa, objID, idx))
		}
		key := strings.Join(keyParts, "|")
		groups[key] = append(groups[key], objID)
	}

	var results []Result
	for key, groupObjects := range groups {
		row := make(map[string]string, len(headers))

		// Set group-by fields from key
		keyParts := strings.Split(key, "|")
		for i, gb := range stmt.GroupBy {
			// Find the column with this field name
			for j, col := range stmt.Columns {
				if fa, ok := col.Expr.(FieldAccess); ok {
					colName := fa.Field
					if fa.IsBuiltin {
						colName = "@" + colName
					}
					if colName == gb {
						if i < len(keyParts) {
							row[headers[j]] = keyParts[i]
						}
					}
				}
			}
		}

		// Evaluate aggregate columns
		for i, col := range stmt.Columns {
			if _, already := row[headers[i]]; already {
				continue
			}
			val := evalAggregateExpr(col.Expr, groupObjects, idx)
			row[headers[i]] = val
		}

		results = append(results, Result{Values: row})
	}

	// Apply ORDER BY
	if len(stmt.OrderBy) > 0 {
		sortResults(results, stmt.OrderBy)
	}

	// Apply LIMIT
	if stmt.Limit > 0 && len(results) > stmt.Limit {
		results = results[:stmt.Limit]
	}

	return headers, results, nil
}

func buildHeaders(columns []Column) []string {
	headers := make([]string, len(columns))
	for i, col := range columns {
		if col.Alias != "" {
			headers[i] = col.Alias
		} else {
			headers[i] = exprName(col.Expr)
		}
	}
	return headers
}

func exprName(expr Expr) string {
	switch e := expr.(type) {
	case FieldAccess:
		if e.IsBuiltin {
			return "@" + e.Field
		}
		if e.Object != "" {
			return e.Object + "." + e.Field
		}
		return e.Field
	case FuncCall:
		return e.Name + "()"
	case StarExpr:
		return "*"
	default:
		return "?"
	}
}

func evalColumnExpr(expr Expr, objID uint64, idx *index.Index) string {
	switch e := expr.(type) {
	case FieldAccess:
		return evalFieldAccess(e, objID, idx)
	case FuncCall:
		// Non-aggregate function
		switch strings.ToLower(e.Name) {
		case "classof":
			return idx.ObjectClassName(objID)
		case "tostring":
			return fmt.Sprintf("[0x%x]", objID)
		case "shallowsize":
			obj := idx.Objects[objID]
			if obj != nil {
				return fmt.Sprintf("%d", obj.ShallowSize)
			}
			return "0"
		}
		return ""
	case StarExpr:
		return fmt.Sprintf("0x%x", objID)
	case StringLit:
		return e.Value
	case NumberLit:
		return e.Value
	default:
		return ""
	}
}

func evalFieldAccess(fa FieldAccess, objID uint64, idx *index.Index) string {
	if fa.IsBuiltin {
		switch fa.Field {
		case "shallowSize":
			obj := idx.Objects[objID]
			if obj != nil {
				return fmt.Sprintf("%d", obj.ShallowSize)
			}
			return "0"
		case "class":
			return idx.ObjectClassName(objID)
		case "objectId":
			return fmt.Sprintf("0x%x", objID)
		}
	}
	// Regular field access - would need instance data parsing
	// For now return a placeholder
	return fmt.Sprintf("<%s>", fa.Field)
}

func evalAggregateExpr(expr Expr, objects []uint64, idx *index.Index) string {
	fc, ok := expr.(FuncCall)
	if !ok {
		if len(objects) > 0 {
			return evalColumnExpr(expr, objects[0], idx)
		}
		return ""
	}

	switch strings.ToLower(fc.Name) {
	case "count":
		return fmt.Sprintf("%d", len(objects))
	case "sum":
		if len(fc.Args) > 0 {
			var total uint64
			for _, objID := range objects {
				val := evalColumnExpr(fc.Args[0], objID, idx)
				n, _ := strconv.ParseUint(val, 10, 64)
				total += n
			}
			return fmt.Sprintf("%d", total)
		}
		return "0"
	case "avg":
		if len(fc.Args) > 0 && len(objects) > 0 {
			var total uint64
			for _, objID := range objects {
				val := evalColumnExpr(fc.Args[0], objID, idx)
				n, _ := strconv.ParseUint(val, 10, 64)
				total += n
			}
			return fmt.Sprintf("%d", total/uint64(len(objects)))
		}
		return "0"
	default:
		return ""
	}
}

func evalBoolExpr(expr Expr, objID uint64, idx *index.Index) (bool, error) {
	switch e := expr.(type) {
	case BinaryExpr:
		left := evalColumnExpr(e.Left, objID, idx)
		right := evalColumnExpr(e.Right, objID, idx)

		switch e.Op {
		case TokEq:
			return left == right, nil
		case TokNeq:
			return left != right, nil
		case TokLt:
			return compareNumeric(left, right) < 0, nil
		case TokGt:
			return compareNumeric(left, right) > 0, nil
		case TokLte:
			return compareNumeric(left, right) <= 0, nil
		case TokGte:
			return compareNumeric(left, right) >= 0, nil
		case TokLike:
			pattern := strings.ReplaceAll(right, "%", ".*")
			matched, _ := regexp.MatchString("^"+pattern+"$", left)
			return matched, nil
		}

	case LogicalExpr:
		leftVal, err := evalBoolExpr(e.Left, objID, idx)
		if err != nil {
			return false, err
		}
		rightVal, err := evalBoolExpr(e.Right, objID, idx)
		if err != nil {
			return false, err
		}
		if e.Op == TokAnd {
			return leftVal && rightVal, nil
		}
		return leftVal || rightVal, nil

	case NotExpr:
		val, err := evalBoolExpr(e.Expr, objID, idx)
		return !val, err

	case IsNullExpr:
		val := evalColumnExpr(e.Expr, objID, idx)
		isNull := val == "" || val == "0" || val == "0x0"
		if e.Negate {
			return !isNull, nil
		}
		return isNull, nil
	}

	return false, fmt.Errorf("cannot evaluate expression as boolean")
}

func compareNumeric(a, b string) int {
	na, errA := strconv.ParseFloat(a, 64)
	nb, errB := strconv.ParseFloat(b, 64)
	if errA != nil || errB != nil {
		return strings.Compare(a, b)
	}
	if na < nb {
		return -1
	}
	if na > nb {
		return 1
	}
	return 0
}

func sortResults(results []Result, orderBy []OrderByClause) {
	sort.Slice(results, func(i, j int) bool {
		for _, ob := range orderBy {
			vi := results[i].Values[ob.Field]
			vj := results[j].Values[ob.Field]

			// Also check with function-style names (e.g., "count()")
			if vi == "" {
				vi = results[i].Values[ob.Field+"()"]
			}
			if vj == "" {
				vj = results[j].Values[ob.Field+"()"]
			}

			cmp := compareNumeric(vi, vj)
			if cmp == 0 {
				continue
			}
			if ob.Desc {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	})
}

func isAggregateFunc(name string) bool {
	switch strings.ToLower(name) {
	case "count", "sum", "avg", "min", "max":
		return true
	}
	return false
}
