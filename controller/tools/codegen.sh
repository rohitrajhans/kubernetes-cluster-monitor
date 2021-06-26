# Example usage : ./tools/codegen.sh -g /Users/pnaduvat/github.com/code-generator -m github.com/k8s-analytics -p k8s.crd.io:v1alpha1 -w $(pwd)

EINVAL=22

show_help() {
	echo "
Usage: ./tools/codegen.sh -g [ code generator path ] -m [ module name ] -p [ package list ] -w [ workspace root ]

	-g : path to cloned https://github.com/kubernetes/code-generator
	-m : module name which is present in the go.mod
	-p : packages to be build, Eg. k8s.crd.io:v1alpha1
	-w : workspace root
	-h : display help

the command needs to be executed from workspace root
		"
}

while getopts g:m:w:p:h flag
do
    case "${flag}" in
        g) gen=${OPTARG};;
        m) module=${OPTARG};;
        p) pkg=${OPTARG};;
        w) wsroot=${OPTARG};;
		h) help=1
    esac
done

if [[ ! -z $help ]]; then
	show_help
	exit $EINVAL
fi

if [[ -z $gen || -z $module || -z $pkg || -z $wsroot ]]; then
	show_help
	exit $EINVAL
fi

show_input() {
	echo "------------------------------------------------------------"
	echo "using these arguments for generation"
	echo "generator: $gen"
	echo "module: $module"
	echo "package: $pkg"
	echo "workspace root: $wsroot"
	echo "------------------------------------------------------------"
}

if [[ ! -f "$gen/generate-groups.sh" ]]; then
	echo "generate-groups.sh not present in generator path $gen"
	exit $EINVAL
fi

if [[ ! -f "$gen/hack/boilerplate.go.txt" ]]; then
	echo "boilerplate.go.txt not present in generator path $gen"
	exit $EINVAL
fi

cd $wsroot/$module

if [[ ! -f "go.mod" ]]; then
	echo "go.mod not found in module directory $wsroot/$module check the module path"
fi

pkgpath=`echo $pkg | awk -F':' '{print $1"/"$2}'`
pkgpath="$wsroot/$module/pkg/apis/$pkgpath"

if [[ ! -f "$pkgpath/doc.go" || ! -f "$pkgpath/register.go" || ! -f "$pkgpath/types.go" ]]; then
	echo "one of doc.go, register.go, types.go not found in module path $pkgpath"
	exit $EINVAL
fi

show_input

$gen/generate-groups.sh all $module/pkg/client $module/pkg/apis $pkg -h $gen/hack/boilerplate.go.txt -o $wsroot --go-header-file $gen/hack/boilerplate.go.txt
