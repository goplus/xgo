//go:build !darwin

/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package projectdriver

import (
	"debug/elf"
	"debug/pe"
	"fmt"
	"os"
	"runtime"
)

func validateHostExecutable(input *os.File, _ string) error {
	switch runtime.GOOS {
	case "linux":
		file, err := elf.NewFile(input)
		if err != nil {
			return fmt.Errorf("driver output is not a Linux executable: %w", err)
		}
		expected := linuxMachine(runtime.GOARCH)
		if (file.Type != elf.ET_EXEC && file.Type != elf.ET_DYN) || file.Entry == 0 || expected != elf.EM_NONE && file.Machine != expected {
			_ = file.Close()
			return fmt.Errorf("driver output is not a Linux %s executable", runtime.GOARCH)
		}
		return file.Close()
	case "windows":
		file, err := pe.NewFile(input)
		if err != nil {
			return fmt.Errorf("driver output is not a Windows executable: %w", err)
		}
		characteristics, expected := file.Characteristics, windowsMachine(runtime.GOARCH)
		if characteristics&pe.IMAGE_FILE_EXECUTABLE_IMAGE == 0 || characteristics&pe.IMAGE_FILE_DLL != 0 || expected != 0 && file.Machine != expected {
			_ = file.Close()
			return fmt.Errorf("driver output is not a Windows %s executable", runtime.GOARCH)
		}
		return file.Close()
	default:
		return nil
	}
}

func linuxMachine(arch string) elf.Machine {
	switch arch {
	case "386":
		return elf.EM_386
	case "amd64":
		return elf.EM_X86_64
	case "arm":
		return elf.EM_ARM
	case "arm64":
		return elf.EM_AARCH64
	case "loong64":
		return elf.EM_LOONGARCH
	case "mips", "mipsle", "mips64", "mips64le":
		return elf.EM_MIPS
	case "ppc64", "ppc64le":
		return elf.EM_PPC64
	case "riscv64":
		return elf.EM_RISCV
	case "s390x":
		return elf.EM_S390
	default:
		return elf.EM_NONE
	}
}

func windowsMachine(arch string) uint16 {
	switch arch {
	case "386":
		return pe.IMAGE_FILE_MACHINE_I386
	case "amd64":
		return pe.IMAGE_FILE_MACHINE_AMD64
	case "arm":
		return pe.IMAGE_FILE_MACHINE_ARMNT
	case "arm64":
		return pe.IMAGE_FILE_MACHINE_ARM64
	default:
		return 0
	}
}
