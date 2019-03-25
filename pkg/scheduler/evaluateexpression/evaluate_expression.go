package evaluateexpression

import (
	"errors"
	"fmt"
	"github.com/Knetic/govaluate"
	"k8s.io/api/core/v1"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var parserRegex, _ = regexp.Compile("(node|pod)\\.(annotations|labels)\\[\\s*['\"](.+?)['\"]\\s*]")

var functions = map[string]govaluate.ExpressionFunction{
	"strlen": func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return float64(0), errors.New("strlen requires 1 argument")
		}
		switch args[0].(type) {
		case string:
			return (float64)(len(args[0].(string))), nil
		default:
			return float64(0), nil
		}
	},
	"round": func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return float64(0), errors.New("round requires 1 argument")
		}
		switch args[0].(type) {
		case float64:
			return math.Round(args[0].(float64)), nil
		default:
			return float64(0), nil
		}
	},
	"floor": func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return float64(0), errors.New("floor requires 1 argument")
		}
		switch args[0].(type) {
		case float64:
			return math.Floor(args[0].(float64)), nil
		default:
			return float64(0), nil
		}
	},
	"ceil": func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return float64(0), errors.New("ceil requires 1 argument")
		}
		switch args[0].(type) {
		case float64:
			return math.Ceil(args[0].(float64)), nil
		default:
			return float64(0), nil
		}
	},
	"toFixed": func(args ...interface{}) (interface{}, error) {
		if len(args) != 2 {
			return float64(0), errors.New("toFixed requires 2 arguments")
		}
		switch args[0].(type) {
		case float64:
		default:
			return float64(0), errors.New("invalid first argument of toFixed")
		}
		switch args[1].(type) {
		case float64:
		default:
			return float64(0), errors.New("invalid second argument of toFixed")
		}
		return fmt.Sprintf("%."+fmt.Sprint(args[1])+"f", args[0]), nil
	},

	"number": func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return float64(0), errors.New("number requires 1 argument")
		}
		switch args[0].(type) {
		case float64:
			return args[0], nil
		case string:
			if args[0].(string) == "" {
				return float64(0), nil
			}
			float, err := strconv.ParseFloat(args[0].(string), 64)
			if err == nil {
				return float, nil
			}
			return 0, err
		case bool:
			if args[0].(bool) {
				return float64(1), nil
			} else {
				return float64(0), nil
			}
		}
		return 0, errors.New("invalid type")
	},

	"string": func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return float64(0), errors.New("string requires 1 argument")
		}
		switch args[0].(type) {
		case float64:
			return fmt.Sprint(args[0]), nil
		case string:
			return args[0], nil
		case bool:
			return fmt.Sprint(args[0]), nil
		}
		return 0, errors.New("invalid type")
	},

	"bool": func(args ...interface{}) (interface{}, error) {
		if len(args) != 1 {
			return float64(0), errors.New("bool requires 1 argument")
		}
		switch args[0].(type) {
		case float64:
			return math.Abs(args[0].(float64)*10000000000) > 1, nil
		case string:
			value := args[0].(string)
			return !(value == "" || value == "false" || value == "0" || value == "null" || value == "nil"), nil
		case bool:
			return args[0], nil
		}
		return 0, errors.New("invalid type")
	},
}

func EvaluateExpression(expression string, node *v1.Node, pod *v1.Pod) (value interface{}, err error) {
	submatch := parserRegex.FindAllStringSubmatch(expression, -1)
	params := make(map[string]interface{})
	for i := 0; i < len(submatch); i++ {
		paramName := "param" + strconv.Itoa(i)
		switch submatch[i][1] {
		case "node":
			switch submatch[i][2] {
			case "annotations":
				params[paramName] = parse(node.GetAnnotations(), submatch[i][3])
			case "labels":
				params[paramName] = parse(node.GetLabels(), submatch[i][3])
			}
		case "pod":
			switch submatch[i][2] {
			case "annotations":
				params[paramName] = parse(pod.GetAnnotations(), submatch[i][3])
			case "labels":
				params[paramName] = parse(pod.GetLabels(), submatch[i][3])
			}
		}

		expression = strings.Replace(expression, submatch[i][0], paramName, -1)
	}

	goexp, err := govaluate.NewEvaluableExpressionWithFunctions(expression, functions)
	if err != nil {
		return nil, err
	}
	result, err := goexp.Evaluate(params)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func parse(strings map[string]string, s string) interface{} {
	value, ok := strings[s]
	if !ok {
		return ""
	}
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}
	float, err := strconv.ParseFloat(value, 64)
	if err == nil {
		return float
	}
	return value
}
