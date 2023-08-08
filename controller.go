package gosocket

import (
	"encoding/json"
	"reflect"
	"strings"
)

// Go language can't create instance from class name dynamically
// therefore I use a map to reflect the string to the class here
// Just use the Router function to add a new route with a new controller
// 由于Go无法动态创建类型，故使用map将字符串映射到类型
var (
	controllerMap map[string]IController
)

// The max payload length is 16kb
// 最大请求长度
const (
	kMaxPayloadLength = (1 << 14) - 1
)

func init() {
	controllerMap = make(map[string]IController)
}

// Router Register a controller to its corresponding name
// 注册Controller以及对应的名字
func Router(controllerName string, controller IController) {
	controllerMap[controllerName] = controller
}

// These are all the status codes that embedded in ResponseBody
// 返回状态码
const (
	StatusSuccess = Status(iota)

	StatusError         = Status(4) //Format error, always return with a message
	StatusInternalError = Status(5) //Unknown internal error, always found with a log record under runtime directory
)

type Status uint8

// ResponseBody the format of the server response
// 返回数据
type ResponseBody struct {
	Status  Status    `json:"status"`            // Status code
	Message string    `json:"message,omitempty"` // Message to the client to show the exact error
	Data    IRespData `json:"data"`              // Real data sent to the client
}

type IRespData interface{}

// IController the interface that all controllers should be implemented
// Controller接口，用于多态
type IController interface {
	Init(user IUser, data []byte)
	GetActionParamMap() map[string]interface{}
	BeforeAction(paramStr string)
	AfterAction(data *ResponseBody)
}

// Controller the base class of all controllers
// Controller基类，用于共同的属性和方法
type Controller struct {
	User IUser
	Data []byte
}

func (controller *Controller) Init(user IUser, data []byte) {
	controller.User = user
	controller.Data = data
}

// BeforeAction run before the action
// action之前执行
func (controller *Controller) BeforeAction(paramStr string) {}

// AfterAction run after the action
// action之后执行
func (controller *Controller) AfterAction(data *ResponseBody) {}

func ProcessPayload(user IUser, payloadType string, payload string) (response *ResponseBody) {
	return ProcessPayloadWithData(user, payloadType, payload, nil)
}

// ProcessPayloadWithData process the request payload
// This function will match the request to a certain action under the controller by reflecting
func ProcessPayloadWithData(user IUser, payloadType string, payload string, data []byte) (response *ResponseBody) {
	defer func() {
		var message = "Internal BackEnd error"
		var status = StatusError
		if r := recover(); r != nil {
			//If an error implemented the `IUserError` interface, then it is a user customize error
			//Just read the message from `ShowError` function
			//如果是用户自定义错误，直接返回错误内容
			if userError, ok := r.(IUserError); ok {
				message = userError.ShowError()
			} else {
				//All the other errors need to be logged
				//其他错误需要记录日志
				err := getRecoverError(r)
				TcpApp.Log.Error(err)
				status = StatusInternalError
			}
			response = nil
		}
		//There must be a response. So if there is no response from the action, generate one
		//如果没有返回，提示开小差
		if response == nil {
			response = &ResponseBody{
				Status:  status,
				Message: message,
			}
		}
	}()
	if len(payload) > kMaxPayloadLength {
		raiseError("length of payload exceeds the max length")
	}
	//Parse the payload type
	//解析type
	strs := strings.Split(payloadType, ".")
	if len(strs) < 2 {
		raiseError("payload type should be in the format of `controller.action`")
	}
	controllerName := strings.ToLower(strs[0])
	actionName := strs[1]
	//Get type of the controller
	controllerPtr := controllerMap[controllerName]
	if controllerPtr == nil {
		raiseError("controller:" + controllerName + "not exist")
	}
	controllerReflectVal := reflect.ValueOf(controllerPtr)
	controllerType := reflect.Indirect(controllerReflectVal).Type()

	//Get controller
	vc := reflect.New(controllerType)
	execController, ok := vc.Interface().(IController)
	if !ok {
		panic("controller is not IController")
	}
	//Initialize controller
	execController.Init(user, data)

	//Get action&param map from child controller
	paramMap := execController.GetActionParamMap()
	if paramMap == nil {
		raiseError("Failed to find ActionParam map for controller:" + controllerName)
	}
	paramPtr := paramMap[actionName]
	if paramPtr == nil {
		raiseError("Failed to find corresponding param in ActionParam map for action:" + actionName + " in controller:" + controllerName)
	}
	paramReflectVal := reflect.ValueOf(paramPtr)
	paramType := reflect.Indirect(paramReflectVal).Type()
	//Get the struct type of payload
	paramVal := reflect.New(paramType)
	paramInt := paramVal.Interface()
	//Only if payload exists
	if len(payload) > 0 {
		//Decode payload into param
		err := json.Unmarshal([]byte(payload), paramInt)
		if err != nil {
			raiseError("Failed to decode payload into param for action:" + actionName + " in controller:" + controllerName)
		}
	}
	//Initialize response body
	response = &ResponseBody{}
	//Default error
	response.Data = struct{}{}
	response.Status = StatusError
	responseVal := reflect.ValueOf(response)
	//find action by name
	method := vc.MethodByName(actionName)
	if !method.IsValid() {
		raiseError("Action:" + actionName + " in controller:" + controllerName + " not found")
	}
	//before hook
	execController.BeforeAction(payload)
	//run action
	method.Call([]reflect.Value{paramVal, responseVal})
	//after hook
	execController.AfterAction(response)
	return
}
