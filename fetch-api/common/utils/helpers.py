from common.utils.system import read_file


def get_app_version(version_file: str):
    try:
        return read_file(version_file, type=None).strip()

    except:
        print(f'{version_file} not found, traces will not have version info.')
        return 'unknown'
