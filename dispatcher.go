package jsonrpc

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
	"unicode"
)

var (
	invalidJSONRequest = &ErrorObject{Code: -32600, Message: "invalid json request"}
	internalError      = &ErrorObject{Code: -32603, Message: "internal error"}
)

func invalidMethod(method string) error {
	return &ErrorObject{Code: -32601, Message: fmt.Sprintf("The method %s does not exist/is not available", method)}
}

func invalidArguments(method string) error {
	return &ErrorObject{Code: -32602, Message: fmt.Sprintf("invalid arguments to %s", method)}
}

type serviceData struct {
	sv      reflect.Value
	funcMap map[string]*funcData
}

type funcData struct {
	inNum int
	reqt  []reflect.Type
	fv    reflect.Value
	isDyn bool
}

func (f *funcData) numParams() int {
	return f.inNum - 1
}

// Dispatcher handles jsonrpc requests
type Dispatcher struct {
	logger     *log.Logger
	serviceMap map[string]*serviceData
}

func NewDispatcher(logger *log.Logger) *Dispatcher {
	return &Dispatcher{logger: logger}
}

func (d *Dispatcher) getFnHandler(req Request) (*serviceData, *funcData, error) {
	callName := strings.SplitN(req.Method, "_", 2)
	if len(callName) != 2 {
		return nil, nil, invalidMethod(req.Method)
	}

	serviceName, funcName := callName[0], callName[1]

	service, ok := d.serviceMap[serviceName]
	if !ok {
		return nil, nil, invalidMethod(req.Method)
	}
	fd, ok := service.funcMap[funcName]
	if !ok {
		return nil, nil, invalidMethod(req.Method)
	}
	return service, fd, nil
}

type wsConn interface {
	WriteMessage(b []byte) error
}

func (d *Dispatcher) Handle(reqBody []byte) ([]byte, error) {
	var req Request
	if err := json.Unmarshal(reqBody, &req); err != nil {
		return nil, invalidJSONRequest
	}

	// its a normal query that we handle with the dispatcher
	resp, err := d.handleReq(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (d *Dispatcher) handleReq(req Request) ([]byte, error) {
	d.logger.Printf("[DEBUG] request: method=%s, id=%s", req.Method, req.ID)

	service, fd, err := d.getFnHandler(req)
	if err != nil {
		return nil, err
	}

	inArgs := make([]reflect.Value, fd.inNum)

	// add service
	inArgs[0] = service.sv

	// decode function input params from request
	typs := fd.reqt[1:]
	inputs := make([]interface{}, len(typs))
	for i := 0; i < len(typs); i++ {
		val := reflect.New(typs[i])
		inputs[i] = val.Interface()
		inArgs[i+1] = val.Elem()
	}

	if err := json.Unmarshal(req.Params, &inputs); err != nil {
		return nil, invalidArguments(req.Method)
	}

	output := fd.fv.Call(inArgs)
	err = getError(output[1])
	if err != nil {
		return nil, d.internalError(req.Method, err)
	}

	var data []byte
	res := output[0].Interface()
	if res != nil {
		data, err = json.Marshal(res)
		if err != nil {
			return nil, d.internalError(req.Method, err)
		}
	}

	resp := Response{
		ID:      req.ID,
		JSONRPC: "2.0",
		Result:  data,
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		return nil, d.internalError(req.Method, err)
	}
	return respBytes, nil
}

func (d *Dispatcher) internalError(method string, err error) error {
	d.logger.Printf("[ERROR] failed to dispatch: method=%s, err=%v", method, err)
	return internalError
}

func (d *Dispatcher) Register(serviceName string, service interface{}) {
	if d.serviceMap == nil {
		d.serviceMap = map[string]*serviceData{}
	}
	if serviceName == "" {
		panic("jsonrpc: serviceName cannot be empty")
	}

	st := reflect.TypeOf(service)
	if st.Kind() == reflect.Struct {
		panic(fmt.Sprintf("jsonrpc: service '%s' must be a pointer to struct", serviceName))
	}

	funcMap := make(map[string]*funcData)
	for i := 0; i < st.NumMethod(); i++ {
		mv := st.Method(i)
		if mv.PkgPath != "" {
			// skip unexported methods
			continue
		}

		name := lowerCaseFirst(mv.Name)
		funcName := serviceName + "_" + name
		fd := &funcData{
			fv: mv.Func,
		}
		var err error
		if fd.inNum, fd.reqt, err = validateFunc(funcName, fd.fv, true); err != nil {
			panic(fmt.Sprintf("jsonrpc: %s", err))
		}

		// check if last item is a pointer
		last := fd.reqt[fd.numParams()]
		if last.Kind() == reflect.Ptr {
			fd.isDyn = true
		}

		funcMap[name] = fd
	}

	d.serviceMap[serviceName] = &serviceData{
		sv:      reflect.ValueOf(service),
		funcMap: funcMap,
	}

}

func validateFunc(funcName string, fv reflect.Value, isMethod bool) (inNum int, reqt []reflect.Type, err error) {
	if funcName == "" {
		err = fmt.Errorf("funcName cannot be empty")
		return
	}

	ft := fv.Type()
	if ft.Kind() != reflect.Func {
		err = fmt.Errorf("function '%s' must be a function instead of %s", funcName, ft)
		return
	}

	inNum = ft.NumIn()
	outNum := ft.NumOut()

	// make sure output arguments have two parameters and the second one is an error
	if outNum != 2 {
		err = fmt.Errorf("unexpected number of output arguments in the function '%s': %d. Expected 2", funcName, outNum)
		return
	}
	if !isErrorType(ft.Out(1)) {
		err = fmt.Errorf("unexpected type for the second return value of the function '%s': '%s'. Expected '%s'", funcName, ft.Out(1), errt)
		return
	}

	reqt = make([]reflect.Type, inNum)
	for i := 0; i < inNum; i++ {
		reqt[i] = ft.In(i)
	}
	return
}

var errt = reflect.TypeOf((*error)(nil)).Elem()

func isErrorType(t reflect.Type) bool {
	return t.Implements(errt)
}

func getError(v reflect.Value) error {
	if v.IsNil() {
		return nil
	}
	return v.Interface().(error)
}

func lowerCaseFirst(str string) string {
	for i, v := range str {
		return string(unicode.ToLower(v)) + str[i+1:]
	}
	return ""
}
