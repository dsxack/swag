package swag

import (
	"errors"
	"fmt"
	"github.com/dave/dst"
	"strings"
)

// ErrFailedConvertPrimitiveType Failed to convert for swag to interpretable type
var ErrFailedConvertPrimitiveType = errors.New("swag property: failed convert primitive type")

type propertyName struct {
	SchemaType string
	ArrayType  string
	CrossPkg   string
}

type propertyNewFunc func(schemeType string, crossPkg string) propertyName

func newArrayProperty(schemeType string, crossPkg string) propertyName {
	return propertyName{
		SchemaType: "array",
		ArrayType:  schemeType,
		CrossPkg:   crossPkg,
	}
}

func newProperty(schemeType string, crossPkg string) propertyName {
	return propertyName{
		SchemaType: schemeType,
		ArrayType:  "string",
		CrossPkg:   crossPkg,
	}
}

func convertFromSpecificToPrimitive(typeName string) (string, error) {
	typeName = strings.ToUpper(typeName)
	switch typeName {
	case "TIME", "OBJECTID", "UUID":
		return "string", nil
	case "DECIMAL":
		return "number", nil
	}
	return "", ErrFailedConvertPrimitiveType
}

func parseFieldSelectorExpr(astTypeSelectorExpr *dst.Ident, parser *Parser, propertyNewFunc propertyNewFunc) propertyName {
	pathParts := strings.Split(astTypeSelectorExpr.Path, "/")
	pkgName := pathParts[len(pathParts)-1]
	typeName := astTypeSelectorExpr.Name

	if primitiveType, err := convertFromSpecificToPrimitive(typeName); err == nil {
		return propertyNewFunc(primitiveType, "")
	}

	if pkgName != "" {
		if typeDefinitions, ok := parser.TypeDefinitions[pkgName][typeName]; ok {
			if expr, ok := typeDefinitions.Type.(*dst.Ident); ok && expr.Path != "" {
				if primitiveType, err := convertFromSpecificToPrimitive(expr.Name); err == nil {
					return propertyNewFunc(primitiveType, "")
				}
			}
			parser.ParseDefinition(pkgName, typeName, typeDefinitions)
			return propertyNewFunc(typeName, pkgName)
		}
		if actualPrimitiveType, isCustomType := parser.CustomPrimitiveTypes[typeName]; isCustomType {
			return propertyName{SchemaType: actualPrimitiveType, ArrayType: actualPrimitiveType}
		}
	}
	return propertyName{SchemaType: "string", ArrayType: "string"}
}

// getPropertyName returns the string value for the given field if it exists
// allowedValues: array, boolean, integer, null, number, object, string
func getPropertyName(expr dst.Expr, parser *Parser) (propertyName, error) {
	if astTypeSelectorExpr, ok := expr.(*dst.Ident); ok && astTypeSelectorExpr.Path != "" {
		return parseFieldSelectorExpr(astTypeSelectorExpr, parser, newProperty), nil
	}

	// check if it is a custom type
	typeName := fmt.Sprintf("%v", expr)
	if actualPrimitiveType, isCustomType := parser.CustomPrimitiveTypes[typeName]; isCustomType {
		return propertyName{SchemaType: actualPrimitiveType, ArrayType: actualPrimitiveType}, nil
	}

	if astTypeIdent, ok := expr.(*dst.Ident); ok {
		name := astTypeIdent.Name
		schemeType := TransToValidSchemeType(name)
		return propertyName{SchemaType: schemeType, ArrayType: schemeType}, nil
	}

	if ptr, ok := expr.(*dst.StarExpr); ok {
		return getPropertyName(ptr.X, parser)
	}

	if astTypeArray, ok := expr.(*dst.ArrayType); ok { // if array
		if _, ok := astTypeArray.Elt.(*dst.StructType); ok {
			return propertyName{SchemaType: "array", ArrayType: "object"}, nil
		}
		return getArrayPropertyName(astTypeArray, parser), nil
	}

	if _, ok := expr.(*dst.MapType); ok { // if map
		return propertyName{SchemaType: "object", ArrayType: "object"}, nil
	}

	if _, ok := expr.(*dst.StructType); ok { // if struct
		return propertyName{SchemaType: "object", ArrayType: "object"}, nil
	}

	if _, ok := expr.(*dst.InterfaceType); ok { // if interface{}
		return propertyName{SchemaType: "object", ArrayType: "object"}, nil
	}
	return propertyName{}, errors.New("not supported" + fmt.Sprint(expr))
}

func getArrayPropertyName(astTypeArray *dst.ArrayType, parser *Parser) propertyName {
	if astTypeArrayExpr, ok := astTypeArray.Elt.(*dst.Ident); ok && astTypeArrayExpr.Path != "" {
		return parseFieldSelectorExpr(astTypeArrayExpr, parser, newArrayProperty)
	}
	if astTypeArrayExpr, ok := astTypeArray.Elt.(*dst.StarExpr); ok {
		if astTypeArraySel, ok := astTypeArrayExpr.X.(*dst.Ident); ok && astTypeArraySel.Path != "" {
			return parseFieldSelectorExpr(astTypeArraySel, parser, newArrayProperty)
		}
		if astTypeArrayIdent, ok := astTypeArrayExpr.X.(*dst.Ident); ok {
			name := TransToValidSchemeType(astTypeArrayIdent.Name)
			return propertyName{SchemaType: "array", ArrayType: name}
		}
	}
	itemTypeName := TransToValidSchemeType(fmt.Sprintf("%s", astTypeArray.Elt))
	if actualPrimitiveType, isCustomType := parser.CustomPrimitiveTypes[itemTypeName]; isCustomType {
		itemTypeName = actualPrimitiveType
	}
	return propertyName{SchemaType: "array", ArrayType: itemTypeName}
}
