package errors

import (
    "fmt"
    "net/http"
)

type Error struct {
    Code int    `json:"code"`
    Msg  string `json:"msg"`
}

func _build(code int, defval string, custom ...string) Error {
    msg := defval
    if len(custom) > 0 {
        msg = custom[0]
    }

    return Error{
        Code: code,
        Msg:  msg,
    }
}

func ErrSwitch(err error) Error {
    if err == nil {
        return _build(http.StatusOK, "success")
    }else if IsParamError(err) ||IsBadRequestError(err) ||IsClientError(err) {
        return _build(http.StatusBadRequest, err.Error())
    }else if IsNotFoundError(err) {
        return _build(http.StatusNotFound, err.Error())
    }else if IsForbiddenError(err)  {
        return _build(http.StatusForbidden, err.Error())
    }else if IsDBError(err) || IsServerError(err) {
        return _build(http.StatusInternalServerError, err.Error())
    }else{
        return _build(432, err.Error())
    }
}


func (e Error) Error() string {
    return fmt.Sprintf("code:%d msg:%s", e.Code, e.Msg)
}


