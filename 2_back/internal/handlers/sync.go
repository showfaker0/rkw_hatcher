package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"rkw_hatcher/internal/wiki"
)

type syncCommitItem struct {
	No          uint64   `json:"no"`
	Name        string   `json:"name"`
	EvoChain    []string `json:"evoChain"`
	EggGroupIDs []uint64 `json:"eggGroupIds"`
	IconURL     string   `json:"iconUrl"`
}

type syncCommitReq struct {
	Items []syncCommitItem `json:"items"`
}

func (a *API) loadEggNameMap() (map[uint64]string, error) {
	rows, err := a.DB.Query(`SELECT id, name FROM egg_groups`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[uint64]string{}
	for rows.Next() {
		var id uint64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		m[id] = name
	}
	return m, nil
}

func (a *API) existingSpeciesNames() (map[string]bool, error) {
	rows, err := a.DB.Query(`SELECT name FROM species`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		m[name] = true
	}
	return m, nil
}

// POST /api/species/sync/preview — 爬取 BWIKI，返回库中尚不存在的最终体候选
func (a *API) syncSpeciesPreview(c *gin.Context) {
	eggMap, err := a.loadEggNameMap()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	exist, err := a.existingSpeciesNames()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	all, err := wiki.FetchFinalForms(eggMap)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "爬取失败: " + err.Error()})
		return
	}
	candidates := make([]wiki.Candidate, 0)
	for _, it := range all {
		if exist[it.Name] {
			continue
		}
		candidates = append(candidates, it)
	}
	c.JSON(http.StatusOK, gin.H{
		"totalFetched": len(all),
		"existing":     len(exist),
		"newCount":     len(candidates),
		"candidates":   candidates,
		"source":       "BWIKI Module:PetData (CC BY-NC-SA 4.0)",
	})
}

// POST /api/species/sync/commit — 仅新增勾选的精灵，不改动已有记录
func (a *API) syncSpeciesCommit(c *gin.Context) {
	var req syncCommitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未选择任何精灵"})
		return
	}

	exist, err := a.existingSpeciesNames()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tx, err := a.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	inserted := 0
	skipped := 0
	for _, it := range req.Items {
		name := strings.TrimSpace(it.Name)
		if name == "" {
			continue
		}
		if exist[name] {
			skipped++
			continue
		}
		chain := it.EvoChain
		if len(chain) == 0 {
			chain = []string{name}
		}
		evoJSON, _ := json.Marshal(chain)
		var icon any
		if it.IconURL != "" {
			icon = it.IconURL
		}
		var no any
		if it.No > 0 {
			no = it.No
		}
		res, err := tx.Exec(`INSERT INTO species (`+"`no`"+`, name, icon_url, evo_chain, notes) VALUES (?, ?, ?, ?, NULL)`, no, name, icon, evoJSON)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": name + ": " + err.Error()})
			return
		}
		id, _ := res.LastInsertId()
		if err := a.replaceSpeciesEggGroups(tx, uint64(id), it.EggGroupIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": name + " 蛋组: " + err.Error()})
			return
		}
		exist[name] = true
		inserted++
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"inserted": inserted, "skipped": skipped})
}
