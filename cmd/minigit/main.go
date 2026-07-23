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

type Index map[string]string

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

	var added []string
	for _, arg := range args {
		if err := walkFiles(arg, func(path string) error {
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

	var staged, modified, missing []string
	for path, hash := range index {
		current, err := fileHash(path)
		if errors.Is(err, os.ErrNotExist) {
			missing = append(missing, path)
			continue
		}
		if err != nil {
			return err
		}
		if current == hash {
			staged = append(staged, path)
		} else {
			modified = append(modified, path)
		}
	}

	untracked, err := findUntracked(index)
	if err != nil {
		return err
	}

	printList("En index", staged)
	printList("Modificados despues de add", modified)
	printList("Eliminados del directorio", missing)
	printList("Sin seguimiento", untracked)
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

func walkFiles(root string, fn func(path string) error) error {
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
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
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == repoDir {
				return filepath.SkipDir
			}
			return nil
		}
		cleanPath, err := normalizePath(path)
		if err != nil {
			return err
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
