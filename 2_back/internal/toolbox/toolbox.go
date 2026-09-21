package toolbox

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	baseURL    = "https://roco.gptvip.chat"
	cdnHeadFmt = "https://rococdn.gptvip.chat/webp/pets/head/%d.webp"
)

var httpClient = &http.Client{Timeout: 25 * time.Second}

// Candidate 同步预览候选（仅用于新增，不改已有图鉴）
type Candidate struct {
	No            uint64   `json:"no"`
	Name          string   `json:"name"`
	EvoChain      []string `json:"evoChain"`
	EggGroupIDs   []uint64 `json:"eggGroupIds"`
	EggGroupNames []string `json:"eggGroupNames"`
	IconURL       string   `json:"iconUrl"`
}

type catalogPage struct {
	OK         bool          `json:"ok"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
	TotalCount int           `json:"total_count"`
	Results    []catalogItem `json:"results"`
}

type catalogItem struct {
	CatalogID    int           `json:"catalog_id"`
	CatalogNo    string        `json:"catalog_no"`
	DisplayName  string        `json:"display_name"`
	IsFinalStage bool          `json:"is_final_stage"`
	Forms        []catalogForm `json:"forms"`
}

type catalogForm struct {
	BaseID             int        `json:"base_id"`
	DisplayName        string     `json:"display_name"`
	SelectorLabel      string     `json:"selector_label"`
	FormLabel          string     `json:"form_label"`
	AvatarURL          string     `json:"avatar_url"`
	AttributeKeys      []string   `json:"attribute_keys"`
	AttributeLabels    []string   `json:"attribute_labels"`
	EggGroupNames      []string   `json:"egg_group_names"`
	EvolutionChainText string     `json:"evolution_chain_text"`
	EvolutionEntries   []evoEntry `json:"evolution_entries"`
	Stage              int        `json:"stage"`
	StageLabel         string     `json:"stage_label"`
	IsDefaultForm      bool       `json:"is_default_form"`
	IsLeaderForm       bool       `json:"is_leader_form"`
	StatChartMax       int        `json:"stat_chart_max"`
	StatRows           []statRow  `json:"stat_rows"`
	TypeRelations      typeRel    `json:"type_relations"`
}

type evoEntry struct {
	Name       string `json:"name"`
	StageIndex int    `json:"stage_index"`
}

type statRow struct {
	Key   string  `json:"key"`
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

type typeRel struct {
	Counter     []string `json:"counter"`
	CounteredBy []string `json:"countered_by"`
	Resist      []string `json:"resist"`
	ResistedBy  []string `json:"resisted_by"`
}

func getJSON(path string, query url.Values, dest any) error {
	u := baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 400 * time.Millisecond)
		}
		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "rkw_hatcher/1.0")
		req.Header.Set("Referer", baseURL+"/")
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("工具箱 HTTP %d", resp.StatusCode)
			continue
		}
		if err := json.Unmarshal(body, dest); err != nil {
			return err
		}
		return nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("工具箱请求失败")
	}
	return lastErr
}

func fetchCatalogPage(page int, specials, q string) (*catalogPage, error) {
	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	query.Set("shiny", "all")
	if specials != "" {
		query.Set("specials", specials)
	}
	if q != "" {
		query.Set("q", q)
	}
	var pg catalogPage
	if err := getJSON("/api/pet-attribute-catalog", query, &pg); err != nil {
		return nil, err
	}
	if !pg.OK {
		return nil, fmt.Errorf("工具箱返回失败")
	}
	return &pg, nil
}

func fetchAllCatalog() ([]catalogItem, error) {
	// 全量图鉴：低阶可能没进 specials=final，回推进化链必须能看到它们。
	first, err := fetchCatalogPage(1, "", "")
	if err != nil {
		return nil, err
	}
	out := append([]catalogItem{}, first.Results...)
	if first.TotalPages <= 1 {
		return out, nil
	}

	type fetched struct {
		items []catalogItem
		err   error
	}
	ch := make(chan fetched, first.TotalPages)
	sem := make(chan struct{}, 6)
	var wg sync.WaitGroup
	for p := 2; p <= first.TotalPages; p++ {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			pg, err := fetchCatalogPage(p, "", "")
			if err != nil {
				ch <- fetched{err: fmt.Errorf("第 %d 页: %w", p, err)}
				return
			}
			ch <- fetched{items: pg.Results}
		}(p)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()
	for f := range ch {
		if f.err != nil {
			return nil, f.err
		}
		out = append(out, f.items...)
	}
	return out, nil
}

func defaultForm(it catalogItem) catalogForm {
	for _, f := range it.Forms {
		if f.IsDefaultForm {
			return f
		}
	}
	if len(it.Forms) > 0 {
		return it.Forms[0]
	}
	return catalogForm{DisplayName: it.DisplayName}
}

func parseCatalogNo(s string) uint64 {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	n, _ := strconv.ParseUint(b.String(), 10, 64)
	return n
}

func liveIconURL(avatar string, baseID int) string {
	avatar = strings.TrimSpace(avatar)
	if avatar != "" {
		return avatar
	}
	return stableIconURL("", baseID)
}

func uniqueNames(names []string) []string {
	out := make([]string, 0, len(names))
	seen := map[string]bool{}
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}

func splitChainText(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, "->", "→")
	return uniqueNames(strings.Split(s, "→"))
}

func evoChainOf(f catalogForm, fallback string) []string {
	if chain := splitChainText(f.EvolutionChainText); len(chain) > 0 {
		return chain
	}
	if len(f.EvolutionEntries) > 0 {
		entries := append([]evoEntry{}, f.EvolutionEntries...)
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].StageIndex != entries[j].StageIndex {
				return entries[i].StageIndex < entries[j].StageIndex
			}
			return i < j
		})
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name)
		}
		if chain := uniqueNames(names); len(chain) > 0 {
			return chain
		}
	}
	if fallback != "" {
		return []string{fallback}
	}
	return []string{}
}

func stableIconURL(avatar string, baseID int) string {
	avatar = strings.TrimSpace(avatar)
	if avatar != "" {
		if u, err := url.Parse(avatar); err == nil {
			u.RawQuery = ""
			u.Fragment = ""
			return u.String()
		}
		if i := strings.Index(avatar, "?"); i >= 0 {
			return avatar[:i]
		}
		return avatar
	}
	if baseID > 0 {
		return fmt.Sprintf(cdnHeadFmt, baseID)
	}
	return ""
}

func eggKey(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, "组")
	if name == "龙" {
		return "巨龙"
	}
	return name
}

func mapEggGroups(remoteNames []string, eggNameByID map[uint64]string) (ids []uint64, names []string) {
	idByKey := map[string]uint64{}
	nameByKey := map[string]string{}
	for id, name := range eggNameByID {
		k := eggKey(name)
		idByKey[k] = id
		nameByKey[k] = name
	}
	seen := map[uint64]bool{}
	for _, raw := range remoteNames {
		k := eggKey(raw)
		id, ok := idByKey[k]
		if !ok || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
		names = append(names, nameByKey[k])
	}
	return ids, names
}

func appearanceLabel(f catalogForm) string {
	lab := strings.TrimSpace(f.FormLabel)
	if lab != "" && lab != "默认形态" {
		return lab
	}
	return ""
}

func formBlob(f catalogForm) string {
	return f.DisplayName + "\n" + f.FormLabel + "\n" + f.SelectorLabel
}

// skipAltForm 首领化 / 污染 / 突变不是可配种图鉴行（各赛季同一套关键词）。
func skipAltForm(f catalogForm) bool {
	if f.IsLeaderForm {
		return true
	}
	t := formBlob(f)
	return strings.Contains(t, "首领") || skipPollutedForm(f)
}

func skipPollutedForm(f catalogForm) bool {
	t := formBlob(f)
	return strings.Contains(t, "被污染") || strings.Contains(t, "突变")
}

// skipSyncForm 图鉴同步：只要本尊最终体。换了展示名的是首领化/特殊形态，样子变体仍同名。
func skipSyncForm(it catalogItem, f catalogForm) bool {
	if skipAltForm(f) {
		return true
	}
	return renamedDisplay(it.DisplayName, f.DisplayName)
}

func renamedDisplay(parent, disp string) bool {
	parent = strings.TrimSpace(parent)
	disp = strings.TrimSpace(disp)
	return parent != "" && disp != "" && disp != parent
}

func keepMaxStage(it catalogItem) int {
	m := 0
	kept := false
	for _, f := range it.Forms {
		if skipSyncForm(it, f) {
			continue
		}
		kept = true
		if f.Stage > m {
			m = f.Stage
		}
	}
	if kept {
		return m
	}
	for _, f := range it.Forms {
		if f.Stage > m {
			m = f.Stage
		}
	}
	return m
}

// isPreEvoItem 工具箱常把整条进化链标成最终体：图鉴号连续且下一只阶段更高，则本只仍是低阶。
func isPreEvoItem(it catalogItem, byNo map[uint64]catalogItem) bool {
	if !it.IsFinalStage {
		return true
	}
	no := parseCatalogNo(it.CatalogNo)
	if no == 0 {
		return false
	}
	nxt, ok := byNo[no+1]
	if !ok {
		return false
	}
	return keepMaxStage(it) < keepMaxStage(nxt)
}

// inferredChain 工具箱没给 evolution_chain 时，按连续图鉴号+阶段回推低阶。
func inferredChain(it catalogItem, byNo map[uint64]catalogItem) []string {
	name := strings.TrimSpace(it.DisplayName)
	no := parseCatalogNo(it.CatalogNo)
	if name == "" {
		return nil
	}
	names := []string{name}
	for i := 0; i < 4 && no > 1; i++ {
		prev, ok := byNo[no-1]
		if !ok || !isPreEvoItem(prev, byNo) {
			break
		}
		pn := strings.TrimSpace(prev.DisplayName)
		if pn == "" {
			break
		}
		names = append([]string{pn}, names...)
		no--
	}
	return uniqueNames(names)
}

func inferredChainFromItems(matched catalogItem, related []catalogItem) []string {
	byNo := map[uint64]catalogItem{}
	if n := parseCatalogNo(matched.CatalogNo); n > 0 {
		byNo[n] = matched
	}
	for _, it := range related {
		if n := parseCatalogNo(it.CatalogNo); n > 0 {
			byNo[n] = it
		}
	}
	return inferredChain(matched, byNo)
}

// IsStubEvoChain 本地链只有自身：工具箱没给链时的占位，可被回推结果覆盖。
func IsStubEvoChain(name string, chain []string) bool {
	name = strings.TrimSpace(name)
	base, _ := splitLocalName(name)
	if len(chain) == 0 {
		return true
	}
	if len(chain) != 1 {
		return false
	}
	n := strings.TrimSpace(chain[0])
	return n == "" || n == name || n == base
}

// ChainNamesFromForms 详情形态列表抽进化链名（去掉样子后缀）。
func ChainNamesFromForms(forms []SpeciesForm, localName string) []string {
	var names []string
	seen := map[string]bool{}
	for _, f := range forms {
		if f.IsLeader {
			continue
		}
		n := strings.TrimSpace(f.Name)
		if n == "" {
			continue
		}
		base, _ := splitLocalName(n)
		if base != "" {
			n = base
		}
		if seen[n] {
			continue
		}
		seen[n] = true
		names = append(names, n)
	}
	end := strings.TrimSpace(localName)
	if b, _ := splitLocalName(end); b != "" {
		end = b
	}
	return chainEndingWith(names, end)
}

func sameDisplayFormCount(it catalogItem, disp string) int {
	n := 0
	parent := strings.TrimSpace(it.DisplayName)
	for _, f := range it.Forms {
		if skipSyncForm(it, f) {
			continue
		}
		d := strings.TrimSpace(f.DisplayName)
		if d == "" {
			d = parent
		}
		if d == disp {
			n++
		}
	}
	return n
}

// localSpeciesName 对齐本地「一形态一行」：同精灵多样子写成 本名（样子）。
func localSpeciesName(it catalogItem, f catalogForm) string {
	parent := strings.TrimSpace(it.DisplayName)
	disp := strings.TrimSpace(f.DisplayName)
	if disp == "" {
		disp = parent
	}
	if disp == "" {
		return ""
	}
	if disp != parent {
		return disp
	}
	label := appearanceLabel(f)
	if label == "" || label == disp {
		return disp
	}
	if sameDisplayFormCount(it, disp) > 1 {
		return disp + "（" + label + "）"
	}
	return disp
}

func splitLocalName(name string) (base, form string) {
	name = strings.TrimSpace(name)
	if i := strings.LastIndex(name, "（"); i >= 0 && strings.HasSuffix(name, "）") {
		return strings.TrimSpace(name[:i]), strings.TrimSpace(name[i+len("（") : len(name)-len("）")])
	}
	if i := strings.LastIndex(name, "("); i >= 0 && strings.HasSuffix(name, ")") {
		return strings.TrimSpace(name[:i]), strings.TrimSpace(name[i+1 : len(name)-1])
	}
	return name, ""
}

func chainEndingWith(chain []string, name string) []string {
	if name == "" {
		return chain
	}
	if len(chain) == 0 {
		return []string{name}
	}
	if chain[len(chain)-1] == name {
		return chain
	}
	out := append([]string{}, chain[:len(chain)-1]...)
	return append(out, name)
}

func itemMatchesName(it catalogItem, query, formHint string) bool {
	query = strings.TrimSpace(query)
	if query == "" {
		return false
	}
	if strings.TrimSpace(it.DisplayName) == query {
		return true
	}
	for _, f := range it.Forms {
		if strings.TrimSpace(f.DisplayName) == query {
			return true
		}
		if n := localSpeciesName(it, f); n != "" && n == query {
			return true
		}
		lab := appearanceLabel(f)
		if lab != "" && (lab == query || (formHint != "" && lab == formHint)) {
			return true
		}
		sel := strings.TrimSpace(f.SelectorLabel)
		if sel != "" && (sel == query || (formHint != "" && sel == formHint)) {
			return true
		}
	}
	return false
}

// FetchFinalForms 翻页拉全量图鉴，再筛最终体本尊。
// 各赛季同一套规则：只要最终体本尊；排除首领化 / 换名特殊形态 / 污染 / 突变；排除未完成进化。
func FetchFinalForms(eggNameByID map[uint64]string) ([]Candidate, error) {
	items, err := fetchAllCatalog()
	if err != nil {
		return nil, err
	}
	byNo := map[uint64]catalogItem{}
	for _, it := range items {
		if n := parseCatalogNo(it.CatalogNo); n > 0 {
			byNo[n] = it
		}
	}
	out := make([]Candidate, 0, len(items))
	seen := map[string]bool{}
	for _, it := range items {
		if isPreEvoItem(it, byNo) {
			continue
		}
		forms := it.Forms
		if len(forms) == 0 {
			forms = []catalogForm{{DisplayName: it.DisplayName, IsDefaultForm: true}}
		}
		def := defaultForm(it)
		for _, form := range forms {
			if skipSyncForm(it, form) {
				continue
			}
			if !form.IsDefaultForm && form.Stage > 0 && def.Stage > 0 && form.Stage < def.Stage {
				continue
			}
			name := localSpeciesName(it, form)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			chain := evoChainOf(form, strings.TrimSpace(it.DisplayName))
			if inf := inferredChain(it, byNo); len(inf) > len(chain) {
				chain = inf
			}
			chain = chainEndingWith(chain, name)
			eggIDs, eggNames := mapEggGroups(form.EggGroupNames, eggNameByID)
			out = append(out, Candidate{
				No:            parseCatalogNo(it.CatalogNo),
				Name:          name,
				EvoChain:      chain,
				EggGroupIDs:   eggIDs,
				EggGroupNames: eggNames,
				IconURL:       liveIconURL(form.AvatarURL, form.BaseID),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].No != out[j].No {
			return out[i].No < out[j].No
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}
