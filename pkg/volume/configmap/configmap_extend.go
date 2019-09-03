/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package configmap

import "os"

const perm os.FileMode = 0777

func (b *configMapVolumeMounter) ensureDir(dir string) error {
	// stat the directory to ensure its existence
	_, err := os.Lstat(dir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(dir, perm); err != nil {
		return err
	}
	return nil
}
