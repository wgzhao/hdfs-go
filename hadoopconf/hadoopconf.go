// Package hadoopconf provides utilities for reading and parsing Hadoop's xml
// configuration files.
package hadoopconf

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type property struct {
	Name  string `xml:"name"`
	Value string `xml:"value"`
}

type propertyList struct {
	Property []property `xml:"property"`
}

var confFiles = []string{"core-site.xml", "hdfs-site.xml", "mapred-site.xml"}

// defaultConfDir is where the Hadoop packages shipped by most Linux
// distributions (and by Cloudera and Ambari) install their configuration.
const defaultConfDir = "/etc/hadoop/conf"

// HadoopConf represents a map of all the key value configutation
// pairs found in a user's hadoop configuration files.
type HadoopConf map[string]string

// LoadFromEnvironment tries to locate the Hadoop configuration files based on
// the environment, and returns a HadoopConf object representing the parsed
// configuration.
//
// Directories are searched in this order:
//
//   - $HADOOP_CONF_DIR
//   - $HADOOP_HOME/etc/hadoop (the layout used by Hadoop 2.x and later)
//   - $HADOOP_HOME/conf (the older Hadoop 1.x and CDH 5 layout)
//   - /etc/hadoop/conf (the location used by most Linux packages), on
//     non-Windows systems
//
// If no configuration can be found, it returns a nil map. If the configuration
// files exist but there was an error opening or parsing them, that is returned
// as well.
func LoadFromEnvironment() (HadoopConf, error) {
	for _, dir := range confDirs() {
		if conf, err := Load(dir); conf != nil || err != nil {
			return conf, err
		}
	}

	return nil, nil
}

// confDirs returns the candidate configuration directories, in priority order.
func confDirs() []string {
	var dirs []string

	if dir := os.Getenv("HADOOP_CONF_DIR"); dir != "" {
		dirs = append(dirs, dir)
	}

	if home := os.Getenv("HADOOP_HOME"); home != "" {
		dirs = append(dirs, filepath.Join(home, "etc", "hadoop"))
		dirs = append(dirs, filepath.Join(home, "conf"))
	}

	if runtime.GOOS != "windows" {
		dirs = append(dirs, defaultConfDir)
	}

	return dirs
}

// Load returns a HadoopConf object representing configuration from the
// specified path. It will parse core-site.xml, hdfs-site.xml, and
// mapred-site.xml.
//
// If no configuration files could be found, Load returns a nil map. If the
// configuration files exist but there was an error opening or parsing them,
// that is returned as well.
func Load(path string) (HadoopConf, error) {
	var conf HadoopConf

	for _, file := range confFiles {
		pList := propertyList{}
		f, err := ioutil.ReadFile(filepath.Join(path, file))
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return conf, err
		}

		err = xml.Unmarshal(f, &pList)
		if err != nil {
			return conf, fmt.Errorf("%s: %s", path, err)
		}

		if conf == nil {
			conf = make(HadoopConf)
		}

		for _, prop := range pList.Property {
			conf[prop.Name] = prop.Value
		}
	}

	return conf, nil
}

// Namenodes returns the namenode hosts present in the configuration. The
// returned slice will be sorted and deduped. The values are loaded from
// fs.defaultFS (or the deprecated fs.default.name), or fields beginning with
// dfs.namenode.rpc-address.
//
// To handle 'logical' clusters Namenodes will not return any cluster names
// found in dfs.ha.namenodes.<clustername> properties.
//
// If no namenode addresses can befound, Namenodes returns a nil slice.
func (conf HadoopConf) Namenodes() []string {
	nns := make(map[string]bool)
	var clusterNames []string

	for key, value := range conf {
		if strings.Contains(key, "fs.default") {
			nnUrl, _ := url.Parse(value)
			nns[nnUrl.Host] = true
		} else if strings.HasPrefix(key, "dfs.namenode.rpc-address.") {
			nns[value] = true
		} else if strings.HasPrefix(key, "dfs.ha.namenodes.") {
			clusterNames = append(clusterNames, key[len("dfs.ha.namenodes."):])
		}
	}

	for _, cn := range clusterNames {
		delete(nns, cn)
	}

	if len(nns) == 0 {
		return nil
	}

	keys := make([]string, 0, len(nns))
	for k, _ := range nns {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	return keys
}
