package api

import (
	"net/http"

	"github.com/tinyzimmer/kvdi/pkg/apis/kvdi/v1alpha1"
	"github.com/tinyzimmer/kvdi/pkg/util/apiutil"
	"github.com/tinyzimmer/kvdi/pkg/util/common"
	"github.com/tinyzimmer/kvdi/pkg/util/errors"
)

// swagger:route GET /api/users Users getUsers
// Retrieves all the users currently known to kVDI.
// responses:
//   200: usersResponse
//   400: error
//   403: error
func (d *desktopAPI) GetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := d.auth.GetUsers()
	if err != nil {
		apiutil.ReturnAPIError(err, w)
		return
	}
	mfaUsers, err := d.mfa.GetMFAUsers()
	if err != nil {
		apiutil.ReturnAPIError(err, w)
		return
	}
	for _, user := range users {
		if common.StringSliceContains(mfaUsers, user.Name) {
			user.MFAEnabled = true
		}
	}
	apiutil.WriteJSON(users, w)
}

// swagger:operation GET /api/users/{user} Users getUser
// ---
// summary: Retrieve the specified user.
// description: Details include the roles, grants, namespaces, and template patterns for the user.
// parameters:
// - name: user
//   in: path
//   description: The username to retrieve details about
//   type: string
//   required: true
// responses:
//   "200":
//     "$ref": "#/responses/userResponse"
//   "400":
//     "$ref": "#/responses/error"
//   "403":
//     "$ref": "#/responses/error"
//   "404":
//     "$ref": "#/responses/error"
func (d *desktopAPI) GetUser(w http.ResponseWriter, r *http.Request) {
	username := apiutil.GetUserFromRequest(r)
	user, err := d.auth.GetUser(username)
	if err != nil {
		if errors.IsUserNotFoundError(err) {
			apiutil.ReturnAPINotFound(err, w)
			return
		}
		apiutil.ReturnAPIError(err, w)
		return
	}
	if _, err := d.mfa.GetUserSecret(username); err != nil && !errors.IsUserNotFoundError(err) {
		apiutil.ReturnAPIError(err, w)
		return
	} else if err == nil {
		user.MFAEnabled = true
	}
	apiutil.WriteJSON(user, w)
}

// A list of users
// swagger:response usersResponse
type swaggerUsersResponse struct {
	// in:body
	Body []v1alpha1.VDIUser
}

// A single user
// swagger:response userResponse
type swaggerUserResponse struct {
	// in:body
	Body v1alpha1.VDIUser
}
