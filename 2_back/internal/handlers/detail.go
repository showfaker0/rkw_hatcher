package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"rkw_hatcher/internal/toolbox"
)

// GET /api/species/:id/detail — 按本地名字转发工具箱详情（六维/形态/克制）。
// 个性/蛋组不改；若本地进化链仍是「仅自身」占位，则用回推链补上。
func (a *API) getSpeciesDetail(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	s, err := a.fetchSpecies(id)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	detail, err := toolbox.FetchSpeciesDetail(s.Name, s.EvoChain)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	if toolbox.IsStubEvoChain(s.Name, s.EvoChain) {
		if chain := toolbox.ChainNamesFromForms(detail.Forms, s.Name); len(chain) > 1 {
			if evo, e := joinJSONArray(chain); e == nil {
				_, _ = a.DB.Exec(`UPDATE species SET evo_chain=? WHERE id=?`, evo, id)
			}
		}
	}
	c.JSON(http.StatusOK, detail)
}
