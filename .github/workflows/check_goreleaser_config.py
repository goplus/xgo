import os
import subprocess
import yaml

files = [f for f in os.listdir('.') if not f.startswith(
    ".") and f not in ["VERSION"]]
files.sort()
# filter out the files that are ignored by git
files = [f for f in files if subprocess.call(
    ["git", "ls-files", "--error-unmatch", f],
    stdout=subprocess.DEVNULL,
    stderr=subprocess.DEVNULL) == 0]

gorel = yaml.load(open(".goreleaser.yaml", "r"), Loader=yaml.FullLoader)
scfiles = [f["source"] for f in gorel["snapcrafts"][0]["extra_files"]]
scfiles.sort()

failed = False

if files != scfiles:
    failed = True
    print("Files in snapcraft are different from the ones in the repo")
    print("Update .goreleaser.yaml in the snapcraft section:")
    for f in files:
        print(f"      - source: \"{f}\"")
        print(f"        destination: \"{f}\"")

nfpms_files = [f["src"]
               for f in gorel["nfpms"][0]["contents"] if f.get("type") != "symlink"]
nfpms_files.sort()

if files != nfpms_files:
    failed = True
    print("Files in nfpms are different from the ones in the repo")
    print("Update .goreleaser.yaml in the nfpms section:")
    for f in files:
        print(f"      - src: \"{f}\"")
        print(f"        dst: \"/usr/lib/{{{{ .ProjectName }}}}/{f}\"")

# Check archives[0].files
archives_files = gorel["archives"][0].get("files", [])
archives_files.sort()

if files != archives_files:
    failed = True
    print("Files in archives are different from the ones in the repo")
    print("Update .goreleaser.yaml in the archives section:")
    for f in files:
        print(f"      - \"{f}\"")

if failed:
    exit(1)

print(".goreleaser checks passed")
