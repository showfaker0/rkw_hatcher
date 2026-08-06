package wiki

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const apiURL = "https://wiki.biligame.com/rocom/api.php"
const fileURLPrefix = "https://wiki.biligame.com/rocom/Special:FilePath/"

var httpClient = &http.Client{Timeout: 90 * time.Second}

// Candidate 爬取整理后的最终形态候选（尚未入库）
type Candidate struct {
	No            uint64   `json:"no"`
	Name          string   `json:"name"`
	EvoChain      []string `json:"evoChain"`
	EggGroupIDs   []uint64 `json:"eggGroupIds"`
	EggGroupNames []string `json:"eggGroupNames"`
	IconURL       string   `json:"iconUrl"`
	WikiPetID     string   `json:"wikiPetId,omitempty"`
}

type petInfo struct {
	ID          string
	Name        string
	EggIDs      []uint64
	EvoGroup    string
	Form        string // f= 如「首领形态」
	Illust      string // img.il 立绘文件名（无扩展名）
	Head        string // img.hd 头像文件名（无扩展名）
	HandbookNo  uint64 // hb.i = handbook_000339 → 339
}

type evoChain struct {
	ID        string
	Names     []string // 进化顺序（不含首领化）
	MemberIDs []string
	Last      string
	LastID    string
	Illust    string // 最终体立绘
	Head      string // 最终体头像
	LordIDs   []string
}

// FetchFinalForms 从 BWIKI PetData 模块拉取并整理最终体图鉴候选
func FetchFinalForms(eggNameByID map[uint64]string) ([]Candidate, error) {
	mods, err := fetchModules([]string{
		"Module:PetData/Core",
		"Module:PetData/Evolution",
		"Module:PetData/Index",
	})
	if err != nil {
		// 本地缓存兜底（开发/反爬时）
		if m2, err2 := loadLocalPetDataJSON("petdata.json"); err2 == nil {
			mods = m2
		} else {
			return nil, err
		}
	}
	core := pickMod(mods, "PetData/Core")
	evoRaw := pickMod(mods, "PetData/Evolution")
	indexRaw := pickMod(mods, "PetData/Index")
	if len(core) < 100 || len(evoRaw) < 100 {
		return nil, fmt.Errorf("模块内容过短 core=%d evo=%d keys=%v", len(core), len(evoRaw), modKeys(mods))
	}

	pets := parsePets(core)
	evos := parseEvolutions(evoRaw)
	if len(evos) == 0 {
		if m2, err2 := loadLocalPetDataJSON("petdata.json"); err2 == nil {
			core = pickMod(m2, "PetData/Core")
			evoRaw = pickMod(m2, "PetData/Evolution")
			indexRaw = pickMod(m2, "PetData/Index")
			pets = parsePets(core)
			evos = parseEvolutions(evoRaw)
		}
	}
	if len(evos) == 0 {
		return nil, fmt.Errorf("未能解析出进化链（core=%d evo=%d）", len(core), len(evoRaw))
	}
	nameToPets := parseIndex(indexRaw)
	// pet id → 主名称（取 Index 中第一个映射名）
	nameByPet := map[string]string{}
	for name, ids := range nameToPets {
		for _, id := range ids {
			if _, ok := nameByPet[id]; !ok {
				nameByPet[id] = name
			}
		}
	}
	for id, p := range pets {
		if p.Name == "" {
			if n, ok := nameByPet[id]; ok {
				p.Name = n
				pets[id] = p
			}
		}
	}

	inAnyChain := map[string]bool{} // pet id（普通进化链）
	lordIDs := map[string]bool{}    // 首领化形态 id
	out := make([]Candidate, 0, len(evos)+64)
	seenName := map[string]bool{}
	seenPet := map[string]bool{}

	for _, e := range evos {
		for _, stID := range e.MemberIDs {
			inAnyChain[stID] = true
		}
		for _, id := range e.LordIDs {
			lordIDs[id] = true
		}
		if e.Last == "" {
			continue
		}
		// 同一 pet 只收一次；不同地区形态是不同 pet，即使 handbook 编号相同也各自入库
		if e.LastID != "" && seenPet[e.LastID] {
			continue
		}
		if seenName[e.Last] {
			continue
		}
		// 首领化不属于成年体最终形态
		if p, ok := pets[e.LastID]; ok && (isBossForm(p) || isMutationForm(p)) {
			continue
		}
		if isMutationName(e.Last) {
			continue
		}
		if lordIDs[e.LastID] {
			continue
		}
		if e.LastID != "" {
			seenPet[e.LastID] = true
		}
		seenName[e.Last] = true
		c := Candidate{
			Name:      e.Last,
			EvoChain:  e.Names,
			WikiPetID: e.LastID,
		}
		if p, ok := pets[e.LastID]; ok {
			c.EggGroupIDs = append([]uint64{}, p.EggIDs...)
			c.No = p.HandbookNo
			c.IconURL = buildFileURL(pickIconBase(p.Head, p.Illust, e.Head, e.Illust))
		} else {
			c.IconURL = buildFileURL(pickIconBase(e.Head, e.Illust))
		}
		if c.No == 0 {
			c.No = petIDNumber(e.LastID)
		}
		fillEggNames(&c, eggNameByID)
		out = append(out, c)
	}

	// 未出现在任何进化链 / 首领化中的精灵：自身即最终体（排除首领、突变外观）
	for id, p := range pets {
		if inAnyChain[id] || lordIDs[id] || isBossForm(p) || isMutationForm(p) || seenPet[id] {
			continue
		}
		name := displayPetName(p.Name, "", p.Form)
		if name == "" {
			name = nameByPet[id]
		}
		if name == "" || seenName[name] {
			continue
		}
		seenPet[id] = true
		seenName[name] = true
		c := Candidate{
			Name:        name,
			EvoChain:    []string{name},
			WikiPetID:   id,
			EggGroupIDs: append([]uint64{}, p.EggIDs...),
			No:          p.HandbookNo,
			IconURL:     buildFileURL(pickIconBase(p.Head, p.Illust)),
		}
		if c.No == 0 {
			c.No = petIDNumber(id)
		}
		fillEggNames(&c, eggNameByID)
		out = append(out, c)
	}

	resolveIconURLs(out)

	sort.Slice(out, func(i, j int) bool {
		if out[i].No != out[j].No {
			return out[i].No < out[j].No
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func petIDNumber(id string) uint64 {
	id = strings.TrimPrefix(id, "pet_")
	n, _ := strconv.ParseUint(id, 10, 64)
	return n
}

func isBossForm(p petInfo) bool {
	return p.Form == "首领形态" || strings.Contains(p.Form, "首领")
}

// isMutationForm 突变/异色炫彩外观数据，不是独立可配种图鉴
func isMutationForm(p petInfo) bool {
	if strings.Contains(p.Form, "突变") {
		return true
	}
	if strings.Contains(p.Name, "突变") {
		return true
	}
	return false
}

func isMutationName(name string) bool {
	return strings.Contains(name, "突变")
}

// displayPetName 地区/季节等形态用完整展示名区分（编号可相同）
func displayPetName(name, title, form string) string {
	base := strings.TrimSpace(title)
	if base == "" {
		base = strings.TrimSpace(name)
	}
	if base == "" {
		return ""
	}
	form = strings.TrimSpace(form)
	if form == "" || strings.Contains(base, "（") || strings.Contains(base, "(") {
		return base
	}
	return base + "（" + form + "）"
}

func pickIconBase(bases ...string) string {
	for _, b := range bases {
		b = strings.TrimSpace(b)
		if b != "" {
			return b
		}
	}
	return ""
}

func buildFileURL(base string) string {
	if base == "" {
		return ""
	}
	base = strings.TrimSuffix(base, ".png")
	base = strings.TrimSuffix(base, ".PNG")
	// Special:FilePath 对空格/下划线等价；不要过度 escape，避免裂图
	return fileURLPrefix + strings.ReplaceAll(base, " ", "_") + ".png"
}

// resolveIconURLs 批量把 Special:FilePath 解析成 patchwiki CDN 直链，减少浏览器跳转裂图
func resolveIconURLs(list []Candidate) {
	bases := make([]string, 0, len(list))
	seen := map[string]bool{}
	for _, c := range list {
		base := fileBaseFromURL(c.IconURL)
		if base == "" || seen[base] {
			continue
		}
		seen[base] = true
		bases = append(bases, base)
	}
	if len(bases) == 0 {
		return
	}
	cdn := map[string]string{}
	const batch = 40
	for i := 0; i < len(bases); i += batch {
		j := i + batch
		if j > len(bases) {
			j = len(bases)
		}
		part, err := fetchImageURLs(bases[i:j])
		if err != nil {
			continue
		}
		for k, v := range part {
			cdn[k] = v
		}
	}
	if len(cdn) == 0 {
		return
	}
	for i := range list {
		base := fileBaseFromURL(list[i].IconURL)
		if u := cdn[base]; u != "" {
			list[i].IconURL = u
		}
	}
}

func fileBaseFromURL(u string) string {
	if u == "" {
		return ""
	}
	if i := strings.LastIndex(u, "/"); i >= 0 {
		u = u[i+1:]
	}
	u, _ = url.PathUnescape(u)
	u = strings.TrimSuffix(u, ".png")
	u = strings.TrimSuffix(u, ".PNG")
	return strings.ReplaceAll(u, " ", "_")
}

func fetchImageURLs(bases []string) (map[string]string, error) {
	titles := make([]string, 0, len(bases))
	for _, b := range bases {
		titles = append(titles, "File:"+strings.ReplaceAll(b, " ", "_")+".png")
	}
	q := url.Values{}
	q.Set("action", "query")
	q.Set("prop", "imageinfo")
	q.Set("iiprop", "url")
	q.Set("titles", strings.Join(titles, "|"))
	q.Set("format", "json")
	full := apiURL + "?" + q.Encode()

	req, err := http.NewRequest(http.MethodGet, full, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json,text/plain,*/*")
	req.Header.Set("Referer", "https://wiki.biligame.com/rocom/")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("imageinfo HTTP %d", resp.StatusCode)
	}
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, err
	}
	out := map[string]string{}
	pages, _ := root["query"].(map[string]any)["pages"].(map[string]any)
	for _, raw := range pages {
		p, _ := raw.(map[string]any)
		title, _ := p["title"].(string) // 文件:JL xxx.png
		infos, _ := p["imageinfo"].([]any)
		if len(infos) == 0 {
			continue
		}
		info, _ := infos[0].(map[string]any)
		imgURL, _ := info["url"].(string)
		if imgURL == "" {
			continue
		}
		base := title
		base = strings.TrimPrefix(base, "文件:")
		base = strings.TrimPrefix(base, "File:")
		base = strings.TrimSuffix(base, ".png")
		base = strings.TrimSuffix(base, ".PNG")
		base = strings.ReplaceAll(base, " ", "_")
		out[base] = imgURL
	}
	return out, nil
}

func loadLocalPetDataJSON(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var root map[string]any
	if err := json.Unmarshal(b, &root); err != nil {
		return nil, err
	}
	out := map[string]string{}
	pages, _ := root["query"].(map[string]any)["pages"].(map[string]any)
	for _, raw := range pages {
		p, _ := raw.(map[string]any)
		title, _ := p["title"].(string)
		revs, _ := p["revisions"].([]any)
		if len(revs) == 0 {
			continue
		}
		rev, _ := revs[0].(map[string]any)
		content, _ := rev["*"].(string)
		if content == "" || title == "" {
			continue
		}
		out[title] = content
		if strings.HasPrefix(title, "模块:") {
			out["Module:"+strings.TrimPrefix(title, "模块:")] = content
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("本地 petdata.json 无可用模块")
	}
	return out, nil
}

func pickMod(mods map[string]string, suffix string) string {
	for _, prefix := range []string{"Module:", "模块:"} {
		if s := mods[prefix+suffix]; s != "" {
			return s
		}
	}
	for k, v := range mods {
		if strings.HasSuffix(k, suffix) {
			return v
		}
	}
	return ""
}

func modKeys(mods map[string]string) []string {
	keys := make([]string, 0, len(mods))
	for k := range mods {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func fillEggNames(c *Candidate, eggNameByID map[uint64]string) {
	c.EggGroupNames = make([]string, 0, len(c.EggGroupIDs))
	for _, id := range c.EggGroupIDs {
		if n, ok := eggNameByID[id]; ok {
			c.EggGroupNames = append(c.EggGroupNames, n)
		} else {
			c.EggGroupNames = append(c.EggGroupNames, strconv.FormatUint(id, 10))
		}
	}
}

func fetchModules(titles []string) (map[string]string, error) {
	q := url.Values{}
	q.Set("action", "query")
	q.Set("prop", "revisions")
	q.Set("rvprop", "content")
	q.Set("rvslots", "main")
	q.Set("titles", strings.Join(titles, "|"))
	q.Set("format", "json")
	full := apiURL + "?" + q.Encode()

	var lastErr error
	waits := []time.Duration{0, 8 * time.Second, 16 * time.Second, 28 * time.Second}
	for attempt, wait := range waits {
		if wait > 0 {
			time.Sleep(wait)
		}
		req, err := http.NewRequest(http.MethodGet, full, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "application/json,text/plain,*/*")
		req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
		req.Header.Set("Referer", "https://wiki.biligame.com/rocom/")
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
		if resp.StatusCode == 567 {
			lastErr = fmt.Errorf("wiki 反爬限制 HTTP 567 (attempt %d)", attempt+1)
			continue
		}
		if resp.StatusCode != 200 {
			lastErr = fmt.Errorf("wiki HTTP %d", resp.StatusCode)
			continue
		}

		var root map[string]any
		if err := json.Unmarshal(body, &root); err != nil {
			return nil, err
		}
		out := map[string]string{}
		pages, _ := root["query"].(map[string]any)["pages"].(map[string]any)
		for _, raw := range pages {
			p, _ := raw.(map[string]any)
			title, _ := p["title"].(string)
			revs, _ := p["revisions"].([]any)
			if len(revs) == 0 {
				continue
			}
			rev, _ := revs[0].(map[string]any)
			content := ""
			if slots, ok := rev["slots"].(map[string]any); ok {
				if main, ok := slots["main"].(map[string]any); ok {
					content, _ = main["*"].(string)
				}
			}
			if content == "" {
				content, _ = rev["*"].(string)
			}
			if content != "" && title != "" {
				// normalize Module: vs 模块:
				out[title] = content
				if strings.HasPrefix(title, "模块:") {
					out["Module:"+strings.TrimPrefix(title, "模块:")] = content
				}
			}
		}
		if len(out) > 0 {
			return out, nil
		}
		lastErr = fmt.Errorf("模块无内容")
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("拉取 PetData 失败")
	}
	return nil, lastErr
}

func fetchModule(title string) (string, error) {
	m, err := fetchModules([]string{title})
	if err != nil {
		return "", err
	}
	if s := m[title]; s != "" {
		return s, nil
	}
	for k, v := range m {
		if strings.HasSuffix(k, strings.TrimPrefix(title, "Module:")) {
			return v, nil
		}
	}
	return "", fmt.Errorf("模块无内容: %s", title)
}

func parsePets(core string) map[string]petInfo {
	out := map[string]petInfo{}
	re := regexp.MustCompile(`(pet_\d+)=\{`)
	matches := re.FindAllStringSubmatchIndex(core, -1)
	for _, m := range matches {
		id := core[m[2]:m[3]]
		body, ok := extractBraced(core, m[1]-1) // '{'
		if !ok {
			continue
		}
		p := petInfo{ID: id}
		n := matchStringField(body, `\bn="([^"]+)"`)
		t := matchStringField(body, `\bt="([^"]+)"`)
		if t == "" {
			t = matchStringField(body, `\btitle="([^"]+)"`)
		}
		if f := matchStringField(body, `\bf="([^"]+)"`); f != "" {
			p.Form = f
		}
		p.Name = displayPetName(n, t, p.Form)
		p.EggIDs = parseUintList(body, `egp=\{([^}]*)\}`)
		if g := matchStringField(body, `evg=\{"([^"]+)"`); g != "" {
			p.EvoGroup = g
		}
		if hb := matchStringField(body, `hb=\{[^}]*i="(handbook_\d+)"`); hb != "" {
			p.HandbookNo = petIDNumber(strings.TrimPrefix(hb, "handbook_"))
		}
		// Core 立绘/头像在 img={ hd=..., il=... }
		if ill := matchStringField(body, `\bil="([^"]+)"`); ill != "" {
			p.Illust = ill
		}
		if hd := matchStringField(body, `\bhd="([^"]+)"`); hd != "" {
			p.Head = hd
		}
		// 兼容旧字段
		if p.Illust == "" {
			if ill := matchStringField(body, `illustration="([^"]+)"`); ill != "" {
				p.Illust = ill
			}
		}
		out[id] = p
	}
	return out
}

func parseEvolutions(raw string) []evoChain {
	var out []evoChain
	re := regexp.MustCompile(`(evo_\d+)=\{`)
	matches := re.FindAllStringSubmatchIndex(raw, -1)
	for _, m := range matches {
		id := raw[m[2]:m[3]]
		body, ok := extractBraced(raw, m[1]-1)
		if !ok {
			continue
		}
		chainBody := extractNamedTable(body, "chain")
		if chainBody == "" {
			continue
		}
		type stage struct {
			display, id, illust, head string
			stage, idx                int
		}
		var stages []stage
		// 逐个提取 chain 内顶层 {...}
		for i := 0; i < len(chainBody); {
			j := strings.IndexByte(chainBody[i:], '{')
			if j < 0 {
				break
			}
			open := i + j
			seg, ok := extractBraced(chainBody, open)
			if !ok {
				break
			}
			rawName := matchStringField(seg, `name="([^"]+)"`)
			rawTitle := matchStringField(seg, `title="([^"]+)"`)
			rawForm := matchStringField(seg, `form="([^"]+)"`)
			display := displayPetName(rawName, rawTitle, rawForm)
			st := stage{
				display: display,
				id:      matchStringField(seg, `id="([^"]+)"`),
				illust:  matchStringField(seg, `illustration="([^"]+)"`),
				head:    matchStringField(seg, `head="([^"]+)"`),
				idx:     len(stages),
			}
			if v := matchStringField(seg, `stage=(\d+)`); v != "" {
				st.stage, _ = strconv.Atoi(v)
			}
			if st.display != "" {
				stages = append(stages, st)
			}
			// move past this brace block
			depth := 0
			k := open
			for ; k < len(chainBody); k++ {
				if chainBody[k] == '{' {
					depth++
				} else if chainBody[k] == '}' {
					depth--
					if depth == 0 {
						k++
						break
					}
				}
			}
			i = k
		}
		if len(stages) == 0 {
			continue
		}
		sort.SliceStable(stages, func(i, j int) bool {
			if stages[i].stage != stages[j].stage {
				if stages[i].stage == 0 {
					return false
				}
				if stages[j].stage == 0 {
					return true
				}
				return stages[i].stage < stages[j].stage
			}
			return stages[i].idx < stages[j].idx
		})
		names := make([]string, 0, len(stages))
		for _, s := range stages {
			names = append(names, s.display)
		}
		last := stages[len(stages)-1]
		memberIDs := make([]string, 0, len(stages))
		for _, s := range stages {
			if s.id != "" {
				memberIDs = append(memberIDs, s.id)
			}
		}
		var lordIDs []string
		if lords := extractNamedTable(body, "lord_branches"); lords != "" {
			reID := regexp.MustCompile(`id="(pet_\d+)"`)
			for _, idm := range reID.FindAllStringSubmatch(lords, -1) {
				lordIDs = append(lordIDs, idm[1])
			}
		}
		out = append(out, evoChain{
			ID:        id,
			Names:     names,
			MemberIDs: memberIDs,
			Last:      last.display,
			LastID:    last.id,
			Illust:    last.illust,
			Head:      last.head,
			LordIDs:   lordIDs,
		})
	}
	return out
}

func parseIndex(raw string) map[string][]string {
	out := map[string][]string{}
	re := regexp.MustCompile(`\["([^"]+)"\]=\{([^}]*)\}`)
	for _, m := range re.FindAllStringSubmatch(raw, -1) {
		name := m[1]
		idsRe := regexp.MustCompile(`"(pet_\d+)"`)
		ids := idsRe.FindAllStringSubmatch(m[2], -1)
		if len(ids) == 0 {
			continue
		}
		list := make([]string, 0, len(ids))
		for _, idm := range ids {
			list = append(list, idm[1])
		}
		out[name] = list
	}
	return out
}

func extractBraced(s string, openIdx int) (string, bool) {
	if openIdx < 0 || openIdx >= len(s) || s[openIdx] != '{' {
		return "", false
	}
	depth := 0
	for i := openIdx; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[openIdx+1 : i], true
			}
		}
	}
	return "", false
}

func extractNamedTable(body, key string) string {
	re := regexp.MustCompile(key + `=\{`)
	loc := re.FindStringIndex(body)
	if loc == nil {
		return ""
	}
	inner, ok := extractBraced(body, loc[1]-1)
	if !ok {
		return ""
	}
	return inner
}

func matchStringField(body, pattern string) string {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(body)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func parseUintList(body, pattern string) []uint64 {
	re := regexp.MustCompile(pattern)
	m := re.FindStringSubmatch(body)
	if len(m) < 2 {
		return nil
	}
	parts := strings.Split(m[1], ",")
	var out []uint64
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.ParseUint(p, 10, 64)
		if err == nil {
			out = append(out, n)
		}
	}
	return out
}
