package template

import (
	"fmt"
	"regexp"
	"strings"

	"k8s.io/kubernetes/pkg/api/meta"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/validation/field"

	"github.com/openshift/origin/pkg/template/api"
	. "github.com/openshift/origin/pkg/template/generator"
	"github.com/openshift/origin/pkg/util"
	"github.com/openshift/origin/pkg/util/stringreplace"
)

var parameterExp = regexp.MustCompile(`\$\{([a-zA-Z0-9\_]+)\}`)

// Processor process the Template into the List with substituted parameters
type Processor struct {
	Generators map[string]Generator
}

// NewProcessor creates new Processor and initializes its set of generators.
func NewProcessor(generators map[string]Generator) *Processor {
	return &Processor{Generators: generators}
}

// Process transforms Template object into List object. It generates
// Parameter values using the defined set of generators first, and then it
// substitutes all Parameter expression occurrences with their corresponding
// values (currently in the containers' Environment variables only).
func (p *Processor) Process(template *api.Template) field.ErrorList {
	templateErrors := field.ErrorList{}

	if fieldError := p.GenerateParameterValues(template); fieldError != nil {
		return append(templateErrors, fieldError)
	}

	itemPath := field.NewPath("item")
	for i, item := range template.Objects {
		idxPath := itemPath.Index(i)
		if obj, ok := item.(*runtime.Unknown); ok {
			// TODO: use runtime.DecodeList when it returns ValidationErrorList
			decodedObj, err := runtime.Decode(runtime.UnstructuredJSONScheme, obj.Raw)
			if err != nil {
				templateErrors = append(templateErrors, field.Invalid(idxPath.Child("objects"), obj, fmt.Sprintf("unable to handle object: %v", err)))
				continue
			}
			item = decodedObj
		}

		newItem, err := p.SubstituteParameters(template.Parameters, item)
		if err != nil {
			templateErrors = append(templateErrors, field.Invalid(idxPath.Child("parameters"), template.Parameters, err.Error()))
		}
		// If an object definition's metadata includes a namespace field, the field will be stripped out of
		// the definition during template instantiation.  This is necessary because all objects created during
		// instantiation are placed into the target namespace, so it would be invalid for the object to declare
		//a different namespace.
		stripNamespace(newItem)
		if err := util.AddObjectLabels(newItem, template.ObjectLabels); err != nil {
			templateErrors = append(templateErrors, field.Invalid(idxPath.Child("labels"),
				template.ObjectLabels, fmt.Sprintf("label could not be applied: %v", err)))
		}
		template.Objects[i] = newItem
	}

	return templateErrors
}

func stripNamespace(obj runtime.Object) {
	// Remove namespace from the item
	if itemMeta, err := meta.Accessor(obj); err == nil && len(itemMeta.GetNamespace()) > 0 {
		itemMeta.SetNamespace("")
		return
	}
	// TODO: allow meta.Accessor to handle runtime.Unstructured
	if unstruct, ok := obj.(*runtime.Unstructured); ok && unstruct.Object != nil {
		if obj, ok := unstruct.Object["metadata"]; ok {
			if m, ok := obj.(map[string]interface{}); ok {
				if _, ok := m["namespace"]; ok {
					m["namespace"] = ""
				}
			}
			return
		}
		if _, ok := unstruct.Object["namespace"]; ok {
			unstruct.Object["namespace"] = ""
			return
		}
	}
}

// AddParameter adds new custom parameter to the Template. It overrides
// the existing parameter, if already defined.
func AddParameter(t *api.Template, param api.Parameter) {
	if existing := GetParameterByName(t, param.Name); existing != nil {
		*existing = param
	} else {
		t.Parameters = append(t.Parameters, param)
	}
}

// GetParameterByName searches for a Parameter in the Template
// based on its name.
func GetParameterByName(t *api.Template, name string) *api.Parameter {
	for i, param := range t.Parameters {
		if param.Name == name {
			return &(t.Parameters[i])
		}
	}
	return nil
}

// SubstituteParameters loops over all values defined in structured
// and unstructured types that are children of item.
//
// Example of Parameter expression:
//   - ${PARAMETER_NAME}
//
func (p *Processor) SubstituteParameters(params []api.Parameter, item runtime.Object) (runtime.Object, error) {
	// Make searching for given parameter name/value more effective
	paramMap := make(map[string]string, len(params))
	for _, param := range params {
		paramMap[param.Name] = param.Value
	}

	stringreplace.VisitObjectStrings(item, func(in string) string {
		for _, match := range parameterExp.FindAllStringSubmatch(in, -1) {
			if len(match) > 1 {
				if paramValue, found := paramMap[match[1]]; found {
					in = strings.Replace(in, match[0], paramValue, 1)
				}
			}
		}
		return in
	})

	return item, nil
}

// GenerateParameterValues generates Value for each Parameter of the given
// Template that has Generate field specified where Value is not already
// supplied.
//
// Examples:
//
// from             | value
// -----------------------------
// "test[0-9]{1}x"  | "test7x"
// "[0-1]{8}"       | "01001100"
// "0x[A-F0-9]{4}"  | "0xB3AF"
// "[a-zA-Z0-9]{8}" | "hW4yQU5i"
// If an error occurs, the parameter that caused the error is returned along with the error message.
func (p *Processor) GenerateParameterValues(t *api.Template) *field.Error {
	for i := range t.Parameters {
		param := &t.Parameters[i]
		if len(param.Value) > 0 {
			continue
		}
		templatePath := field.NewPath("template").Child("parameters").Index(i)
		if param.Generate != "" {
			generator, ok := p.Generators[param.Generate]
			if !ok {
				err := fmt.Errorf("Unknown generator name '%v' for parameter %s", param.Generate, param.Name)
				return field.Invalid(templatePath, param.Generate, err.Error())
			}
			if generator == nil {
				err := fmt.Errorf("template.parameters[%v]: Invalid '%v' generator for parameter %s", i, param.Generate, param.Name)
				return field.Invalid(templatePath, param, err.Error())
			}
			value, err := generator.GenerateValue(param.From)
			if err != nil {
				return field.Invalid(templatePath, param, err.Error())
			}
			param.Value, ok = value.(string)
			if !ok {
				err := fmt.Errorf("template.parameters[%v]: Unable to convert the generated value '%#v' to string for parameter %s", i, value, param.Name)
				return field.Invalid(templatePath, param, err.Error())
			}
		}
		if len(param.Value) == 0 && param.Required {
			err := fmt.Errorf("template.parameters[%v]: parameter %s is required and must be specified", i, param.Name)
			return field.Required(templatePath, err.Error())
		}
	}
	return nil
}
