package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math/rand/v2"
	"os"
	"os/user"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

const GraftVersion = "0003"

// NOTE: As of 17/12/2025, it seems that WriteConfig does not turn path into absolutes

// NOTE: We implement this type so we can
// implkement a custom writer for it
// to take advantage of TeeReader
type transferStatus struct {
	bytesWritten uint64
}

// NOTE: teeReader sends to the writer, everything that it reads
// This allows us to put a file we want to copy on the reader
// and make a struct that satisfies the writer interface
// allowing us to read n amount of bytes from the source file
// and then send those same bytes to a writer
// which in this case its this function
// AAAAAAAAAAAAAAAAAAAND we can keep track of everything
// We should implement a way to do it every n amount of bytes,
// I say every GB
func (tf *transferStatus) Write(data []byte) (int, error) {
	tf.bytesWritten += uint64(len(data))
	// Jump to the beginning of the previous line (assumes we println)
	fmt.Printf("\x1b[1F")
	// Clear the line
	fmt.Printf("\x1b[2K")
	// FIXME
	fmt.Printf("cool i got %f\n", float64(tf.bytesWritten)/1000000000)
	return len(data), nil
}

// NOTE: Would be cool to have an error datatype for grafter
// also maybe change the name from grafter?

// This is the main data type for the individual things we graft
// We should try to not lose info, we could
// save the whole fs.FileInfo but I will refrain for now
// MORSEL: A scion is what we insert into the root
type scion struct {
	// ID, will be similar to stem
	// Based on filename but modified if necessary
	ID string
	// File name
	// We can obtain this from stat
	Stem string
	// Path where it is currently stored
	Path string
	// Path where it was originally stored
	OriginalPath string
	// If its a file or nah
	Dir bool
	// Size of file
	Size int64
	// Mode bits
	Perms fs.FileMode
	// Date it was grafted
	Date time.Time
	// Additional notes
	Note string
	// TODO: Hash
	// TODO: Inputs that generated this file
	// TODO: Command that generated this workflow
	// TODO: Workflow used
	// If it has been grafted yet
	Exists bool
}

type graftage struct {
	// This is where the scion will go
	// NOTE: This will be an option later on
	// MORSEL: This is the stock or rootstock of the graft, where the scion is placed
	StockDir string
	// This is all the files/paths we will graft
	Scions map[string]scion
}

// Also FIXME: Add a check of file name
// TODO: Add checking if the config file exists first and if not create with defaults
// TODO: add an option to make the
// Config is the path where we will load the user config from
// Init function opens the config file and checks if theres anything there
func (g *graftage) Init(config string, stock string) error {
	// Need to init this before anything can be put inside
	g.Scions = make(map[string]scion)
	graftConfigBytes, err := os.ReadFile(config)
	if errors.Is(err, fs.ErrNotExist) {
		log.Println("No config file, using defaults...")
		// Retrieve the value that will be used inside the config
		g.StockDir = stock
	} else if err != nil {
		return err
	} else {
		err = json.Unmarshal(graftConfigBytes, &g)
		if err != nil {
			return err
		}
		if g.StockDir == "" {
			log.Println("Empty graft location, adding default...")
			g.StockDir = stock
		}
	}
	if !filepath.IsAbs(g.StockDir) {
		g.StockDir, err = filepath.Abs(g.StockDir)
		if err != nil {
			return err
		}
	}
	return nil
}

// Writes the graft to the config file
func (g *graftage) WriteConfig(config string) error {
	graftageBytes, err := json.MarshalIndent(g, "", " ")
	if err != nil {
		return err
	}
	err = os.WriteFile(config, graftageBytes, 0o644)
	if err != nil {
		return err
	}
	return nil
}

// Grafts everything in the slice of scions
func (g *graftage) GraftScions(rtScions []scion, move bool, del bool) error {
	// For evaluates the variable we will range over ONCE
	// so we are free to modify the variable and append to it
	// without worrying the loop will extend once again
	// https://go.dev/ref/spec#For_statements
	for i, rs := range rtScions {
		log.Printf("Grafting [%s]\n", rs.OriginalPath)
		// Maybe we could move this to scionsFromArgs
		matchingScion := g.Scions[rs.ID]
		if matchingScion != (scion{}) {
			// Ensure the ID is the same and the original path is the same
			// although we can probably just use the original path
			if matchingScion.ID == rs.ID && matchingScion.OriginalPath == rs.OriginalPath {
				log.Printf("Found [%v] in the user graft...\n",
					rs.ID)

				// Generate a random 4 digit number to append at the end
				r := rand.Uint32() >> 19
				digit := uint64(r)
				digitStr := strconv.FormatUint(digit, 10)
				rs.ID = rs.ID + "-" + digitStr
				if move {
					if rs.Dir {
						// Since its a directory we only need to append
						// Add option to only fetch the indicated ones
						rs.Path = rs.Path + "-" + digitStr
					} else {
						// we need to extract the directory and recreate
						// the full file path
						scion_dir := filepath.Dir(rs.Path)
						rs.Path = filepath.Join(scion_dir, rs.ID)
					}
				}
				log.Printf("Creating copy with ID [%v]", rs.ID)
				// Append at the end of the string
				//rt_scions[j].ID = new_scion_id_t
				// The scion is already in the main graft
			}
		}
		// We wont move it
		// Since rtScions has not been modified
		if !rs.Exists || !move || (move && rs.OriginalPath == rtScions[i].Path) {
			g.Scions[rs.ID] = rs
			continue
		}
		err := rs.copyScionFiles()
		if err != nil {
			return err
		}
		if del {
			err := os.RemoveAll(rs.OriginalPath)
			if err != nil {
				return err
			}
		}
		// Finally add the new member to the graft
		// The range grabs a copy of the variable its ranging over,
		// so it is safe to modify the original while looping
		g.Scions[rs.ID] = rs
	}
	return nil
}

// FIXME: I don't like having the config file here...
// FIXME: Change to use ID and not stem
func (g *graftage) RemoveScions(ids []string, config string) error {
	// if we found anything at all
	found := false
	for _, id := range ids {
		matchingScion := g.Scions[id]
		if matchingScion != (scion{}) {
			log.Printf("Removing [%s] with ID [%s] from the file system...", matchingScion.Path, matchingScion.ID)
			delete(g.Scions, id)
			err := os.RemoveAll(matchingScion.Path)
			if err != nil {
				return err
			}
			found = true
			continue
		}
		log.Printf("Could not find ID [%s]", id)
	}
	if found {
		log.Println("Saving changes to config file...")
		err := g.WriteConfig(config)
		if err != nil {
			return err
		}
		log.Println("Success!")
	}
	return nil
}

// Fetches all the scions that have not been grafted yet
func (g *graftage) FetchMissingFiles(config string) error {
	// Add option to only fetch the indicated ones
	for _, s := range g.Scions {
		if !s.Exists {
			// SECTION: Add missing things in graft
			scionStat, err := os.Stat(s.OriginalPath)
			if err != nil {
				log.Fatal(err)
			}
			// TODO:ID
			s.ID = scionStat.Name()
			// TODO: stem
			s.Stem = scionStat.Name()
			// SECTION: Check if its a dir or a file
			if scionStat.IsDir() {
				s.Dir = true
			}
			// TODO: Size
			s.Size = scionStat.Size()
			// TODO: perms
			s.Perms = scionStat.Mode().Perm()

			s.copyScionFiles()
			g.Scions[s.ID] = s
		}
	}
	err := g.WriteConfig(config)
	if err != nil {
		log.Println("Error writing to config file...")
		log.Fatal(err)
	}
	return nil
}

// Copies the listed ID inside dstPath
func (g *graftage) CopyScionFilesFromIds(ids []string, dstPath string) error {
	err := os.MkdirAll(dstPath, 0o755)
	if err != nil {
		return err
	}
	for _, id := range ids {
		matchedScion := g.Scions[id]
		if matchedScion != (scion{}) {
			dstPathComplete := filepath.Join(dstPath, matchedScion.ID)
			if matchedScion.Dir {
				files, err := os.ReadDir(matchedScion.Path)
				if err != nil {
					return err
				}
				// NOTE: Need to have this because we are placing it
				// inside the directory we pass, so we need to join it
				err = os.MkdirAll(dstPathComplete, matchedScion.Perms)
				if err != nil {
					return err
				}
				_, err = digDirNCreate(files, dstPathComplete, matchedScion.Path)
				if err != nil {
					return err
				}
			} else {
				err = copyFileW(dstPathComplete, matchedScion.Path, matchedScion.Perms)
			}
		}
	}
	return nil
}

// FIXME: Could perhaps make it take a string as an arg to allow for flipping
// original with a path
// Writes the scion to the filesystem aka add it to the graft folder
// FIXME: Change name
func (s *scion) copyScionFiles() error {
	if s.Dir {
		files, err := os.ReadDir(s.OriginalPath)
		if err != nil {
			return err
		}
		// FIXME / DOCS : in the docs we need to mention that
		// he directories inherit the permissions from the main grafted dir
		err = os.MkdirAll(s.Path, s.Perms)
		if err != nil {
			return err
		}
		s.Size, err = digDirNCreate(files, s.Path, s.OriginalPath)
		if err != nil {
			return err
		}
	} else {
		// First we create the directories
		scionDestDir := filepath.Dir(s.Path)
		err := os.MkdirAll(scionDestDir, 0o755)
		if err != nil {
			return err
		}
		err = copyFileW(s.Path, s.OriginalPath, s.Perms)
		if err != nil {
			return err
		}
	}
	return nil
}

// Copy File wrapper
func copyFileW(dst string, src string, perms fs.FileMode) error {
	log.Printf("Copying [%s] to [%s]\n", src, dst)
	// We have to stat once more to check for symlinks
	srcStat, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if srcStat.Mode()&fs.ModeSymlink != 0 {
		log.Printf("Path [%s] is a symlink, preserving...\n", src)
		oldname, err := os.Readlink(src)
		if err != nil {
			return err
		}
		err = os.Symlink(oldname, dst)
		if err != nil {
			return err
		}
		return nil
	}
	// Open file we will read
	srcFi, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFi.Close()
	// Open file where we will write
	dstFi, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, perms)
	if err != nil {
		return err
	}
	defer dstFi.Close()
	io.Copy(dstFi, srcFi)
	// Ensure the permission is respected
	// since sometimes the permission is not
	// added at creation
	os.Chmod(dst, perms)
	return nil
}

// p is the directory where all these entries are in
// FIXME: this whole function needs to be rewritten to incorporate the preamble
// that usually accompanies a call to it
func digDirNCreate(d []fs.DirEntry, dstPath string, srcPath string) (int64, error) {
	var totalSize int64
	for _, f := range d {
		srcFiPath := filepath.Join(srcPath, f.Name())
		srcFiStat, err := os.Stat(srcFiPath)
		if err != nil {
			log.Fatal(err)
		}
		dstFiPath := filepath.Join(dstPath, f.Name())
		if f.IsDir() {
			// Create the dst dir so when its time to copy we can finally do it
			err = os.MkdirAll(dstFiPath, srcFiStat.Mode().Perm())
			if err != nil {
				return 0, err
			}
			newSrcDirEnt, err := os.ReadDir(srcFiPath)
			if err != nil {
				return 0, err
			}
			siz, err := digDirNCreate(newSrcDirEnt, dstFiPath, srcFiPath)
			if err != nil {
				return 0, nil
			}
			totalSize += siz
		} else {
			if srcFiStat.Mode()&fs.ModeSymlink != 0 {
				fmt.Printf("\nSkipping symlink %v\n", srcFiPath)
				os.Exit(1)
				continue
			}
			err := copyFileW(dstFiPath, srcFiPath, srcFiStat.Mode().Perm())
			if err != nil {
				log.Fatal(err)
			}
			totalSize += srcFiStat.Size()
		}
	}
	return totalSize, nil
}

// temp_graft_dir is the directory where these
func scionsFromSlice(rtScions *[]scion, args []string, stock string, userHome string, exist bool, move bool) error {
	for _, v := range args {
		var scionTemp = scion{}
		// XXX: Should we stat first or absolute path first?
		// FIXME: In the future we can add recovering from failed abs
		// and failed stat
		// grafted path
		var scionPath string
		var err error
		if !filepath.IsAbs(v) {
			// We know its not absolute
			// And we only want to handle in a special manner
			// tilde, all other forms like ./
			// we can leave to filepath.Abs()
			if v == "~" {
				scionPath = userHome
			} else if strings.HasPrefix(v, "~/") {
				scionPath = filepath.Join(userHome, v[2:])
			} else {
				scionPath, err = filepath.Abs(v)
				if err != nil {
					log.Fatal(err)
				}
			}
		} else {
			scionPath = v
		}

		// NOTE: Shared values
		// TODO: original path
		scionTemp.OriginalPath = scionPath
		// TODO: note?
		scionTemp.Note = ""
		// TODO: date
		scionTemp.Date = time.Now()

		if !exist {
			scionBase := filepath.Base(v)
			scionTemp.ID = scionBase
			scionTemp.Stem = scionBase
			scionTemp.Path = filepath.Join(stock, scionTemp.Stem)
			scionTemp.Exists = false
		} else {
			// SECTION: stat path
			scionStat, err := os.Stat(scionPath)
			if err != nil {
				log.Fatal(err)
			}
			// TODO:ID
			scionTemp.ID = scionStat.Name()
			// TODO: stem
			scionTemp.Stem = scionStat.Name()
			// TODO: path
			// We want to join this
			// FIXME: Add a runtime option to use a different directory
			if move {
				scionTemp.Path = filepath.Join(stock, scionTemp.Stem)
				fmt.Println(scionTemp.Path)
			} else {
				scionTemp.Path = scionPath
			}
			// SECTION: Check if its a dir or a file
			if scionStat.IsDir() {
				scionTemp.Dir = true
			}
			// TODO: Size
			scionTemp.Size = scionStat.Size()
			// TODO: perms
			scionTemp.Perms = scionStat.Mode().Perm()
			// TODO: Grafted
			// If it wont be grafted we pass a false
			scionTemp.Exists = true
		}
		// TODO: Hash
		// TODO: Inputs that generated this file
		// TODO: Workflow used

		// Finally add to scions to graft later on
		// FIXME: This needs to be a runtime scion and not the same scion we unmarshal
		// so we can properly verify against our graft config
		*rtScions = append(*rtScions, scionTemp)
	}
	return nil
}

type cmdOptions struct {
	version  bool
	list     bool
	fetch    bool
	remove   bool
	exist    bool
	move     bool
	cpy      bool
	copyDir  string
	graftDir string
	del      bool
}

// The way this should works is that we go through the listed files
// and directories with an array of some sort
func main() {
	stdinSlice := make([]string, 0)
	stdinStat, _ := os.Stdin.Stat()
	// Equivalent to !isatty()
	if stdinStat.Mode()&os.ModeCharDevice == 0 {
		in := bufio.NewScanner(os.Stdin)
		in.Split(bufio.ScanWords)
		for in.Scan() {
			stdinSlice = append(stdinSlice, in.Text())
		}
	}
	// SECTION: Init of runtime stuff
	var cmdOpts cmdOptions
	flag.BoolVar(&cmdOpts.version, "v", false, "[verbose] Show the current version of grafter.")
	flag.BoolVar(&cmdOpts.list, "l", false, "[List] List everything in the user graft.")
	flag.BoolVar(&cmdOpts.fetch, "f", false, "[Fetch] Validate and fetch missing files in the user graft.")
	flag.BoolVar(&cmdOpts.remove, "r", false, "[Remove] Remove the listed IDs from the user graft.")
	flag.BoolVar(&cmdOpts.exist, "e", true, "[Exist] If -exist=0 the files will only be grafted to the list.")
	// This flag defaults to true, I think that in general, with the use cases im thinking
	// you will want to make copies of the file
	flag.BoolVar(&cmdOpts.move, "m", true, "[Move] If the file should be moved to the graft directory.")
	flag.BoolVar(&cmdOpts.del, "del", false, "[Delete] Delete the original files after grafting.")
	flag.BoolVar(&cmdOpts.cpy, "c", false, "[Copy] Copy the files of the listed IDs to destdir.")
	flag.StringVar(&cmdOpts.copyDir, "d", "", "[CopyDir] Directory to copy the selected files to.")
	flag.StringVar(&cmdOpts.graftDir, "g", "", "[GraftDir] Directory to place grafted files in.")
	flag.Parse()
	var argList []string
	if len(stdinSlice) == 0 {
		argList = flag.Args()
	} else {
		argList = stdinSlice
	}
	// For tilde expansion
	runtimeUser, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	rtUserHomeDir := runtimeUser.HomeDir
	rtStockDir := cmdOpts.graftDir
	if rtStockDir == "" {
		rtStockDir = filepath.Join(rtUserHomeDir, "Grafts")
	} else if !filepath.IsAbs(rtStockDir) {
		rtStockDir, err = filepath.Abs(rtStockDir)
		if err != nil {
			log.Fatal(err)
		}
	}

	// CAUTION: This needs to be considered when we move to the new file handling
	// empty but has a capacity that is non-zero
	// TODO: Add a function for validating a scion is not in the list
	// TODO: Add a function for updating a scion
	var grafts graftage
	configFile := filepath.Join(rtUserHomeDir, "grafts.json")
	err = grafts.Init(configFile, rtStockDir)
	if err != nil {
		log.Fatal(err)

	}

	if cmdOpts.version {
		fmt.Println("Grafter version:", GraftVersion)
		b, _ := debug.ReadBuildInfo()
		fmt.Println(b.String())
		os.Exit(0)
	} else if cmdOpts.list {
		for _, v := range grafts.Scions {
			fmt.Printf("Date: %-12s | ", v.Date.Format(time.RFC822))
			fmt.Printf("ID: %-12s | ", v.ID)
			fmt.Printf("Original: %s\n", v.OriginalPath)
		}
		os.Exit(0)
	} else if cmdOpts.fetch {
		grafts.FetchMissingFiles(configFile)
		os.Exit(0)
	} else if cmdOpts.cpy {
		if cmdOpts.copyDir == "" {
			log.Println("Empty destdir flag.")
			flag.PrintDefaults()
		}
		err := grafts.CopyScionFilesFromIds(argList, cmdOpts.copyDir)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	} else if cmdOpts.remove {
		err := grafts.RemoveScions(argList, configFile)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}
	if cmdOpts.del && !cmdOpts.move {
		log.Fatal("Must have the move flag enabled when the delete flag is enabled.")
	}

	// SECTION: actual program processing
	var runtimeScions []scion = make([]scion, 0, len(argList))
	// This is where we would add a condition to indicate if in THIS run
	// of the program we would use a different stock directory than
	// the one in the config file/default one

	err = scionsFromSlice(&runtimeScions, argList, rtStockDir, rtUserHomeDir, cmdOpts.exist, cmdOpts.move)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: Add a section validating that the runtime scions are different from the ones in the graftr
	// if we find one thats similar we either update or remove it
	// this can be an option too
	// NOTE: We only need to do operations on the runtime scions and at the end update the
	// json...
	// XXX: We will validate the runtime in its own section so we can give the user the option to
	// remove it of the list or update it, although for now we will just ignore the runtime one
	// We will loop the existing grafts and compare against the runtime scions,
	// as opposed to loop the runtime scions and compare against the
	// existing scions
	// I think this is indicative that we need a better data structure than
	// an array with all the scions loll
	// We could in theory add the looping of the runtime scions here but no,
	// not for now
	err = grafts.GraftScions(runtimeScions, cmdOpts.move, cmdOpts.del)
	if err != nil {
		log.Fatal(err)
	}
	err = grafts.WriteConfig(configFile)
	if err != nil {
		log.Println("Error writing to config file...")
		log.Fatal(err)
	}
	// TODO / FIXME So we need a way to save to a file,
	// Need to create a new syntax for that
	// Probably parsing since I dont want to use json
	// We need to think about the date format
	// solution probably related to the many ways golang gives to represent dates
}
