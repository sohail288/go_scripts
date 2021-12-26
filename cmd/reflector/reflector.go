package main

import (
	"fmt"
	"log"
	"reflect"
	"strings"
)

type MyType struct {
	First  int    `mytag:"first" othertag:"damn"`
	Second string `mytag:"second"`
}

// add validator!
func (mt *MyType) ValidateFirst() error {
	if mt.First != 2 {
		return fmt.Errorf("invalid value for First")
	}
	mt.First = 42
	return nil
}

func (mt *MyType) ValidateSecond() error {
	mt.Second = strings.Trim(mt.Second, " ")
	if len(mt.Second) == 0 {
		return fmt.Errorf("Invalid value for Second")
	}
	return nil
}

func main() {
	myType := MyType{2, " lol "}

	reflectMyType := reflect.ValueOf(myType)

	fmt.Println("Info")
	fmt.Printf("type=%s \r\nvalue=%s\r\n kind=%s\r\noriginal: %s\r\n", reflect.TypeOf(myType), reflectMyType.String(), reflectMyType.Kind(), reflectMyType.Interface())

	fmt.Println("Settability")

	ptrReflect := reflect.ValueOf(&myType)
	fmt.Printf("type=%s \r\nvalue=%s\r\n kind=%s\r\noriginal: %s\r\n", reflect.TypeOf(myType), ptrReflect.String(), ptrReflect.Kind(), ptrReflect.Interface())

	fmt.Printf("Can set non-pointer: %t\r\nCan set ptr: %t\r\n", ptrReflect.CanSet(), ptrReflect.Elem().CanSet())

	fmt.Println("Iteration over fields")
	elem := ptrReflect.Elem()
	elemType := elem.Type()

	if elem.Kind() != reflect.Struct {
		log.Panicln("wrong type?")
	}
	fmt.Printf("Type we have: %s\n", elem.Kind().String())

	// must use the elem num field function since the elem.Type().NumField is not a struct ...
	numFields := elem.NumField()
	fmt.Printf("%s has %d fields\n", elem.Type(), numFields)
	fmt.Printf("%s has %d methods\n", elem.Type(), ptrReflect.NumMethod())

	for i := 0; i < ptrReflect.NumMethod(); i++ {
		m := ptrReflect.Method(i)
		fmt.Printf("%s - %s - %s\n", m.Type(), m.Kind(), m.Interface())
	}

	for i := 0; i < numFields; i++ {
		f := elem.Field(i)
		fmt.Printf("i=%d name=%s type=%s value=%s\n", i, elemType.Field(i).Name, f.Type(), f.Interface())

		validatorName := fmt.Sprintf("Validate%s", elemType.Field(i).Name)
		fmt.Printf("Looking for %s\n", validatorName)

		// depending on ptr receiver or value receiver, so would need to check both?
		// ValidateX = value receiver
		// ApplyX - ptr receiver?
		// can we have multiple validators for a single field? order will be tricky
		// would need to iterate over all methods?
		validatorMethod := ptrReflect.MethodByName(validatorName)
		if validatorMethod.IsValid() {
			out := validatorMethod.Call([]reflect.Value{})
			if len(out) == 0 {
				fmt.Println("wtf?")
			}
			err := out[0]
			if !err.IsNil() {
				log.Fatalf("Got error validation: %s\n", err)
			}

			fmt.Println("applied validation")
			fmt.Println(myType)
		}

		//  this only works on concrete type? not value
		tField := elemType.Field(i)
		for _, f := range []string{"mytag", "othertag"} {
			if tagvalue, ok := tField.Tag.Lookup(f); ok {
				fmt.Printf("%s = %s\n", f, tagvalue)
			}
		}
	}
}
