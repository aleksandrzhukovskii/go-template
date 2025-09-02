package model

import "errors"

var ErrorNoRowsUpdated = errors.New("no rows updated")
var ErrorNoRowsDeleted = errors.New("no rows deleted")
var ErrorNoUpdateParams = errors.New("no parameters were passed to be updated")
