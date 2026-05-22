package files

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// listPageSize returns page and page size from query (?page=1&limit=20 or legacy ?per_page=).
func listPageSize(c *gin.Context) (page int, pageSize int) {
	page = parsePositiveInt(c.DefaultQuery("page", strconv.Itoa(defaultListPage)), defaultListPage)
	if page < 1 {
		page = 1
	}

	limitStr := strings.TrimSpace(c.Query("limit"))
	if limitStr != "" {
		pageSize = parsePositiveInt(limitStr, defaultListPerPage)
	} else {
		pageSize = parsePositiveInt(c.DefaultQuery("per_page", strconv.Itoa(defaultListPerPage)), defaultListPerPage)
	}
	if pageSize > maxListPerPage {
		pageSize = maxListPerPage
	}
	if pageSize < 1 {
		pageSize = defaultListPerPage
	}
	return page, pageSize
}
