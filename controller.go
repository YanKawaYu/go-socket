package gosocket

import (
	"encoding/json"
	"reflect"
	"strings"
)

// 由于Go无法动态创建类型，故使用map将字符串映射到类型
var (
	controllerMap map[string]IController
)

// 最大请求长度，16kb
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

// 返回状态码
const (
	StatusSuccess = Status(iota)

	StatusError         = Status(4) //已知的错误，会返回message给客户端
	StatusInternalError = Status(5) //未知的内部错误，如数据库异常等，会打印日志
)

type Status uint8

// ResponseBody the format of the server response
// 返回数据
type ResponseBody struct {
	Status  Status    `json:"status"`
	Message string    `json:"message,omitempty"`
	Data    IRespData `json:"data"`
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
		var message = "服务器开小差啦~"
		var status = StatusError
		//捕获异常
		if r := recover(); r != nil {
			//如果是用户自定义错误，直接返回错误内容
			if userError, ok := r.(IUserError); ok {
				message = userError.ShowError()
			} else {
				//其他错误需要记录日志
				err := getRecoverError(r)
				TcpApp.Log.Error(err)
				status = StatusInternalError
			}
			response = nil
		}
		//如果没有返回，提示开小差
		if response == nil {
			response = &ResponseBody{
				Status:  status,
				Message: message,
			}
		}
	}()
	//检查payload长度
	if len(payload) > kMaxPayloadLength {
		raiseError("消息太长了哦~")
	}
	//解析type
	strs := strings.Split(payloadType, ".")
	if len(strs) < 2 {
		raiseError("类型应该为controller.action哦~")
	}
	controllerName := strings.ToLower(strs[0])
	actionName := strs[1]
	//获取controller类型
	controllerPtr := controllerMap[controllerName]
	if controllerPtr == nil {
		raiseError("controller:" + controllerName + "不存在哦~")
	}
	controllerReflectVal := reflect.ValueOf(controllerPtr)
	controllerType := reflect.Indirect(controllerReflectVal).Type()

	//获取controller
	vc := reflect.New(controllerType)
	execController, ok := vc.Interface().(IController)
	if !ok {
		panic("controller is not IController")
	}
	//初始化controller
	execController.Init(user, data)

	//初始化调用参数
	paramPtr := execController.GetActionParamMap()[actionName]
	if paramPtr == nil {
		raiseError("controller:" + controllerName + "的action:" + actionName + "不存在哦~")
	}
	paramReflectVal := reflect.ValueOf(paramPtr)
	paramType := reflect.Indirect(paramReflectVal).Type()
	//获取param
	paramVal := reflect.New(paramType)
	paramInt := paramVal.Interface()
	//payload为空字符串的时候不解析
	if len(payload) > 0 {
		//解析载荷,不使用jsons包中的函数，因为要捕捉错误
		err := json.Unmarshal([]byte(payload), paramInt)
		if err != nil {
			raiseError("参数类型错了哦~")
		}
	}
	//初始化返回参数
	response = &ResponseBody{}
	//默认为错误
	response.Data = struct{}{}
	response.Status = StatusError
	responseVal := reflect.ValueOf(response)
	//调用action
	method := vc.MethodByName(actionName)
	if !method.IsValid() {
		raiseError("action不存在哦~")
	}
	//before hook
	execController.BeforeAction(payload)
	//执行action
	method.Call([]reflect.Value{paramVal, responseVal})
	//after hook
	execController.AfterAction(response)
	return
}
