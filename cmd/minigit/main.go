package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const repoDir = ".minigit"
const ignoreFile = ".minigitignore"

var defaultIgnoreRules = IgnoreRules{".git", repoDir}

type Index map[string]string
type IgnoreRules []string

type Commit struct {
	ID        string            `json:"id"`
	Message   string            `json:"message"`
	Parent    string            `json:"parent,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	Files     map[string]string `json:"files"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "init":
		err = initRepo()
	case "add":
		err = add(os.Args[2:])
	case "commit":
		err = commit(os.Args[2:])
	case "log":
		err = showLog()
	case "status":
		err = status()
	case "diff":
		err = diff(os.Args[2:])
	case "checkout":
		err = checkout(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		err = fmt.Errorf("comando desconocido: %s", os.Args[1])
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`Mini-Git en Go

Uso:
  minigit init
  minigit add <archivo|carpeta> [...]
  minigit commit -m "mensaje"
  minigit status
  minigit diff [archivo]
  minigit log
  minigit checkout <commit-id>`)
}

func initRepo() error {
	for _, dir := range []string{objectsDir(), commitsDir()} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	created := false
	if !exists(indexPath()) {
		if err := writeJSON(indexPath(), Index{}); err != nil {
			return err
		}
		created = true
	}
	if !exists(headPath()) {
		if err := os.WriteFile(headPath(), []byte(""), 0644); err != nil {
			return err
		}
		created = true
	}
	if created {
		fmt.Println("Repositorio Mini-Git inicializado en .minigit")
	} else {
		fmt.Println("El repositorio .minigit ya estaba inicializado")
	}
	return nil
}

func add(args []string) error {
	if len(args) == 0 {
		return errors.New("indica al menos un archivo o carpeta")
	}
	if err := ensureRepo(); err != nil {
		return err
	}

	index, err := readIndex()
	if err != nil {
		return err
	}
	rules, err := readIgnoreRules()
	if err != nil {
		return err
	}

	var added []string
	for _, arg := range args {
		if err := walkFiles(arg, rules, func(path string) error {
			hash, err := storeObject(path)
			if err != nil {
				return err
			}
			cleanPath, err := normalizePath(path)
			if err != nil {
				return err
			}
			index[cleanPath] = hash
			added = append(added, cleanPath)
			return nil
		}); err != nil {
			return err
		}
	}

	if err := writeJSON(indexPath(), index); err != nil {
		return err
	}
	sort.Strings(added)
	for _, path := range added {
		fmt.Println("agregado:", path)
	}
	return nil
}

func commit(args []string) error {
	if err := ensureRepo(); err != nil {
		return err
	}

	flags := flag.NewFlagSet("commit", flag.ContinueOnError)
	message := flags.String("m", "", "mensaje del commit")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*message) == "" {
		return errors.New(`usa commit -m "mensaje"`)
	}

	index, err := readIndex()
	if err != nil {
		return err
	}
	if len(index) == 0 {
		return errors.New("no hay archivos agregados al index")
	}

	head, err := readHead()
	if err != nil {
		return err
	}
	files := make(map[string]string, len(index))
	for path, hash := range index {
		files[path] = hash
	}

	c := Commit{
		Message:   strings.TrimSpace(*message),
		Parent:    head,
		CreatedAt: time.Now().UTC(),
		Files:     files,
	}
	c.ID = commitID(c)

	if err := writeJSON(filepath.Join(commitsDir(), c.ID+".json"), c); err != nil {
		return err
	}
	if err := os.WriteFile(headPath(), []byte(c.ID), 0644); err != nil {
		return err
	}

	fmt.Println("commit creado:", c.ID)
	return nil
}

func showLog() error {
	if err := ensureRepo(); err != nil {
		return err
	}
	head, err := readHead()
	if err != nil {
		return err
	}
	if head == "" {
		fmt.Println("No hay commits todavia.")
		return nil
	}

	for id := head; id != ""; {
		c, err := readCommit(id)
		if err != nil {
			return err
		}
		fmt.Printf("commit %s\nFecha: %s\nMensaje: %s\n\n", c.ID, c.CreatedAt.Format(time.RFC3339), c.Message)
		id = c.Parent
	}
	return nil
}

func status() error {
	if err := ensureRepo(); err != nil {
		return err
	}
	index, err := readIndex()
	if err != nil {
		return err
	}

	headCommit, err := readHeadCommit()
	if err != nil {
		return err
	}

	var staged, modified, missing, unchanged []string
	for path, hash := range index {
		current, err := fileHash(path)
		if errors.Is(err, os.ErrNotExist) {
			missing = append(missing, path)
			continue
		}
		if err != nil {
			return err
		}
		if current != hash {
			modified = append(modified, path)
			continue
		}
		if headCommit == nil || headCommit.Files[path] != hash {
			staged = append(staged, path)
		} else {
			unchanged = append(unchanged, path)
		}
	}

	untracked, err := findUntracked(index)
	if err != nil {
		return err
	}

	printList("Cambios preparados para commit", staged)
	printList("Cambios no preparados", modified)
	printList("Eliminados del directorio", missing)
	printList("Archivos sin seguimiento", untracked)
	printList("Sin cambios", unchanged)
	return nil
}

func diff(args []string) error {
	if len(args) > 1 {
		return errors.New("uso: minigit diff [archivo]")
	}
	if err := ensureRepo(); err != nil {
		return err
	}
	index, err := readIndex()
	if err != nil {
		return err
	}

	paths := make([]string, 0, len(index))
	if len(args) == 1 {
		cleanPath, err := normalizePath(args[0])
		if err != nil {
			return err
		}
		if _, ok := index[cleanPath]; !ok {
			return fmt.Errorf("archivo no agregado al index: %s", cleanPath)
		}
		paths = append(paths, cleanPath)
	} else {
		for path := range index {
			paths = append(paths, path)
		}
		sort.Strings(paths)
	}

	changed := false
	for _, path := range paths {
		hash := index[path]
		oldData, err := os.ReadFile(filepath.Join(objectsDir(), hash))
		if err != nil {
			return err
		}
		newData, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			fmt.Printf("diff %s\n- archivo eliminado\n\n", path)
			changed = true
			continue
		}
		if err != nil {
			return err
		}
		if string(oldData) == string(newData) {
			continue
		}
		printDiff(path, string(oldData), string(newData))
		changed = true
	}
	if !changed {
		fmt.Println("No hay diferencias.")
	}
	return nil
}

func checkout(args []string) error {
	if len(args) != 1 {
		return errors.New("uso: minigit checkout <commit-id>")
	}
	if err := ensureRepo(); err != nil {
		return err
	}

	c, err := readCommit(args[0])
	if err != nil {
		return err
	}
	for path, hash := range c.Files {
		data, err := os.ReadFile(filepath.Join(objectsDir(), hash))
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			return err
		}
	}
	if err := writeJSON(indexPath(), Index(c.Files)); err != nil {
		return err
	}
	if err := os.WriteFile(headPath(), []byte(c.ID), 0644); err != nil {
		return err
	}
	fmt.Println("checkout aplicado:", c.ID)
	return nil
}

func walkFiles(root string, rules IgnoreRules, fn func(path string) error) error {
	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("no existe: %s", root)
		}
		return err
	}
	if !info.IsDir() {
		cleanPath, err := normalizePath(root)
		if err != nil {
			return err
		}
		if shouldIgnore(cleanPath, rules) {
			return nil
		}
		return fn(root)
	}
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == repoDir {
				return filepath.SkipDir
			}
			cleanPath, err := normalizePath(path)
			if err != nil {
				return err
			}
			if shouldIgnore(cleanPath, rules) {
				return filepath.SkipDir
			}
			return nil
		}
		cleanPath, err := normalizePath(path)
		if err != nil {
			return err
		}
		if shouldIgnore(cleanPath, rules) {
			return nil
		}
		return fn(path)
	})
}

func storeObject(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	hash := hashBytes(data)
	objectPath := filepath.Join(objectsDir(), hash)
	if !exists(objectPath) {
		if err := os.WriteFile(objectPath, data, 0644); err != nil {
			return "", err
		}
	}
	return hash, nil
}

func fileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return hashBytes(data), nil
}

func hashBytes(data []byte) string {
	sum := sha1.Sum(data)
	return hex.EncodeToString(sum[:])
}

func commitID(c Commit) string {
	var builder strings.Builder
	builder.WriteString(c.Message)
	builder.WriteString(c.Parent)
	builder.WriteString(c.CreatedAt.Format(time.RFC3339Nano))
	keys := make([]string, 0, len(c.Files))
	for path := range c.Files {
		keys = append(keys, path)
	}
	sort.Strings(keys)
	for _, path := range keys {
		builder.WriteString(path)
		builder.WriteString(c.Files[path])
	}
	return hashBytes([]byte(builder.String()))
}

func findUntracked(index Index) ([]string, error) {
	var result []string
	rules, err := readIgnoreRules()
	if err != nil {
		return nil, err
	}
	err = filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == repoDir {
				return filepath.SkipDir
			}
			cleanPath, err := normalizePath(path)
			if err != nil {
				return err
			}
			if shouldIgnore(cleanPath, rules) {
				return filepath.SkipDir
			}
			return nil
		}
		cleanPath, err := normalizePath(path)
		if err != nil {
			return err
		}
		if shouldIgnore(cleanPath, rules) {
			return nil
		}
		if _, ok := index[cleanPath]; !ok {
			result = append(result, cleanPath)
		}
		return nil
	})
	sort.Strings(result)
	return result, err
}

func printList(title string, items []string) {
	sort.Strings(items)
	fmt.Println(title + ":")
	if len(items) == 0 {
		fmt.Println("  (ninguno)")
		return
	}
	for _, item := range items {
		fmt.Println("  " + item)
	}
}

func printDiff(path, oldText, newText string) {
	fmt.Println("diff", path)
	oldLines := splitLines(oldText)
	newLines := splitLines(newText)
	max := len(oldLines)
	if len(newLines) > max {
		max = len(newLines)
	}
	for i := 0; i < max; i++ {
		var oldLine, newLine string
		oldOK := i < len(oldLines)
		newOK := i < len(newLines)
		if oldOK {
			oldLine = oldLines[i]
		}
		if newOK {
			newLine = newLines[i]
		}
		switch {
		case oldOK && newOK && oldLine == newLine:
			fmt.Println("  " + oldLine)
		case oldOK && newOK:
			fmt.Println("- " + oldLine)
			fmt.Println("+ " + newLine)
		case oldOK:
			fmt.Println("- " + oldLine)
		case newOK:
			fmt.Println("+ " + newLine)
		}
	}
	fmt.Println()
}

func splitLines(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func readIgnoreRules() (IgnoreRules, error) {
	rules := append(IgnoreRules{}, defaultIgnoreRules...)
	data, err := os.ReadFile(ignoreFile)
	if errors.Is(err, os.ErrNotExist) {
		return rules, nil
	}
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		rules = append(rules, filepath.ToSlash(line))
	}
	return rules, nil
}

func shouldIgnore(path string, rules IgnoreRules) bool {
	path = filepath.ToSlash(filepath.Clean(path))
	base := filepath.Base(path)
	for _, rule := range rules {
		rule = filepath.ToSlash(strings.TrimSpace(rule))
		if rule == "" {
			continue
		}
		if strings.HasSuffix(rule, "/") {
			dirRule := strings.Trim(rule, "/")
			if path == dirRule || strings.HasPrefix(path, dirRule+"/") {
				return true
			}
			continue
		}
		rule = strings.Trim(rule, "/")
		if path == rule || strings.HasPrefix(path, rule+"/") {
			return true
		}
		if ok, _ := filepath.Match(rule, base); ok {
			return true
		}
		if ok, _ := filepath.Match(rule, path); ok {
			return true
		}
	}
	return false
}

func ensureRepo() error {
	if !exists(repoDir) {
		return errors.New("no existe .minigit; ejecuta minigit init")
	}
	for _, path := range []string{objectsDir(), commitsDir(), indexPath(), headPath()} {
		if !exists(path) {
			return errors.New("repositorio .minigit incompleto; ejecuta minigit init para repararlo")
		}
	}
	return nil
}

func readIndex() (Index, error) {
	var index Index
	if err := readJSON(indexPath(), &index); err != nil {
		return nil, err
	}
	if index == nil {
		index = Index{}
	}
	return index, nil
}

func readCommit(id string) (Commit, error) {
	var c Commit
	err := readJSON(filepath.Join(commitsDir(), id+".json"), &c)
	return c, err
}

func readHeadCommit() (*Commit, error) {
	head, err := readHead()
	if err != nil {
		return nil, err
	}
	if head == "" {
		return nil, nil
	}
	c, err := readCommit(head)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func readHead() (string, error) {
	data, err := os.ReadFile(headPath())
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func writeJSON(path string, value any) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func readJSON(path string, value any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewDecoder(file).Decode(value)
}

func normalizePath(path string) (string, error) {
	rel, err := filepath.Rel(".", path)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Clean(rel)), nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func objectsDir() string {
	return filepath.Join(repoDir, "objects")
}

func commitsDir() string {
	return filepath.Join(repoDir, "commits")
}

func indexPath() string {
	return filepath.Join(repoDir, "index.json")
}

func headPath() string {
	return filepath.Join(repoDir, "HEAD")
}
