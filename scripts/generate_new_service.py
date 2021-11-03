import os, sys
import shutil

SERVICE_NAME_TEMPLATE = "adax"


def replace_in_file(file_name,new_name,git_account):
    with open(file_name, errors='ignore') as f:
        new_text=f.read().replace(SERVICE_NAME_TEMPLATE, new_name)
        if ("go.mod" in file_name) and (git_account!=""):
            new_text = new_text.replace("github.com/alivinco",git_account)


    with open(file_name, "w") as f:
        f.write(new_text)


def rename_file(old_file_path,new_name):
    if SERVICE_NAME_TEMPLATE in old_file_path:
        new_path = old_file_path.replace(SERVICE_NAME_TEMPLATE,new_name)
        os.replace(old_file_path,new_path)
        
        return new_path
    return old_file_path


def rename_files(new_service_name,git_account):
    for root, dirs, files in os.walk("../"+new_service_name):
        for filename in dirs:
            full_path = root+"/"+filename
            full_path = rename_file(full_path,new_service_name)
            print(full_path)

    for root, dirs, files in os.walk("../"+new_service_name):
        for filename in files:
            full_path = root+"/"+filename
            full_path = rename_file(full_path,new_service_name)
            print(full_path)
            replace_in_file(full_path,new_service_name,git_account)



if __name__ == "__main__":
    new_service_name = sys.argv[1]
    if len(sys.argv)> 2 :
        git_acc = sys.argv[2]
    else:
        git_acc = ""
    # debian1 or debian2
    shutil.copytree("./", "../"+new_service_name, ignore=shutil.ignore_patterns('.idea',".git"))
    rename_files(new_service_name,git_acc)