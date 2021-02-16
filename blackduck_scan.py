from ..core import hooks

from ..utils import logger_config
from ..core.utils import (
    print_script_start,
    print_script_end,
    run_command_with_output,
    step_filter,
)
from ..core.workspace_adapters.workspace_adapter_retriever import get_workspace_adapter
from ..core.workspace_adapters.canaveral_workspace_adapter import (
    CanaveralWorkspaceAdapter,
)

import tempfile
import docker
import logging
import os
import requests
from pathlib import Path

logger_config.configure_logging()
logger = logging.getLogger(__name__)


def _get_scan_branch_tag(workspace_adapter: CanaveralWorkspaceAdapter):
    branch = workspace_adapter.branch
    branch = branch.replace("master", "prod")
    branch = branch.replace("mainline", "prod")
    branch = branch.replace("main", "prod")
    branch = branch.replace("pull/", "pre-merge/")
    branch = branch.replace("pr/", "pre-merge/")

    return branch


def _save_docker_image(image_name: str, save_path: Path):
    docker_client = docker.from_env()
    docker_image = docker_client.images.get(image_name)

    logger.info(f"Writing image {image_name} to {str(save_path)}")

    with save_path.open("wb") as image_file:
        for chunk in docker_image.save():
            image_file.write(chunk)


def _process_image(has_scan_target: bool, workspace_adapter: CanaveralWorkspaceAdapter):
    if os.environ.get("BLACKDUCK_TARGET_DOCKER_IMAGE", "0") == "1":
        target_docker_image = workspace_adapter.image_name

        logger.info(f"docker image to be scanned is specified as {target_docker_image}")
        logger.info("pulling image to make sure it is available locally")

        docker_client = docker.from_env()
        image = docker_client.images.pull(target_docker_image)

        image_path = Path(
            f"{tempfile.gettempdir()}/{workspace_adapter.repo}:{workspace_adapter.build_identifier}.tar"
        )

        logger.info(f"Writing image to {str(image_path)}")
        with image_path.open("wb") as image_file:
            for chunk in image.save():
                image_file.write(chunk)

        blackduck_opts = []

        if "BLACKDUCK_OPTS" in os.environ:
            blackduck_opts.append(os.environ["BLACKDUCK_OPTS"])

        blackduck_opts.append(
            f"--detect.blackduck.signature.scanner.paths={str(image_path)}"
        )

        return True, [image_path], blackduck_opts
    else:
        return has_scan_target, [], []


def _process_docker_images_list(
    has_scan_target: False, workspace_adapter: CanaveralWorkspaceAdapter,
):
    if os.environ.get("BLACKDUCK_TARGET_DOCKER_IMAGES_LIST_FILE"):
        images_list_file = Path(os.environ["BLACKDUCK_TARGET_DOCKER_IMAGES_LIST_FILE"])

        if not images_list_file.is_file():
            message = (
                f"Docker Images File {str(images_list_file)} is not there, aborting."
            )
            logger.error(message)
            raise Exception(message)

        created_tars = []
        blackduck_opts = []

        with images_list_file.open("rt") as images:
            for index, line in enumerate(images, start=1):
                logger.info(f"Docker image to be scanned is specified as ${line}")

                saved_image_path = Path(
                    f"{tempfile.gettempdir()}/{workspace_adapter.repo}:{workspace_adapter.build_identifier}-{index}.tar"
                )
                logger.info(f"Saved to {str(saved_image_path)}")
                _save_docker_image(line, saved_image_path)
                created_tars.append(saved_image_path)
                blackduck_opts.append(
                    f"--detect.blackduck.signature.scanner.paths={str(saved_image_path)}"
                )
                has_scan_target = True

        return has_scan_target, created_tars, blackduck_opts

    return has_scan_target, [], []


def _process_source_path(
    has_scan_target: bool, search_depth: str, project_root_folder: str
):
    blackduck_opts = []
    if os.environ.get("BLACKDUCK_TARGET_SOURCE_PATH", "0") == "1":
        target_source_path = Path(project_root_folder)
        logger.info(f"Source scan path specified as {target_source_path}")
        if not target_source_path.is_dir():
            message = f"Source scan path {target_source_path} doesn't exist, aborting."
            logger.error(message)
            raise Exception(message)


        blackduck_opts.append("--blackduck.username=canaveral.svc")
        blackduck_password = os.getenv("BD_HUB_PASSWORD")
        blackduck_opts.append(f"--blackduck.password={blackduck_password}")
        blackduck_opts.append("--blackduck.url=https://blackduck.eng.nutanix.com")
        blackduck_opts.append("--blackduck.trust.cert=true")
        blackduck_opts.append("--detect.project.user.groups=ALL-Engineering")
        blackduck_opts.append(f"--detect.source.path={target_source_path}")
        blackduck_opts.append("--detect.tools=DETECTOR")
        blackduck_opts.append("--detect.detector.search.continue")
        blackduck_opts.append(f"--detect.detector.search.depth={search_depth}")

	if "BLACKDUCK_OPTS" in os.environ:
            blackduck_opts.append(os.environ["BLACKDUCK_OPTS"])

        return True, [], blackduck_opts

    return has_scan_target, [], blackduck_opts


def _process_binary_path(has_scan_target: bool):
    blackduck_opts = []
    created_tars = []
    if os.environ.get("BLACKDUCK_TARGET_BINARY_PATH"):
        target_binary_path = Path(os.environ.get("BLACKDUCK_TARGET_BINARY_PATH"))
        logger.info(f"Build binary scan path is specified as {target_binary_path}")
        if not target_binary_path.is_file():
            message = f"File {target_binary_path} doesn't exist, aborting"
            logger.error(message)
            raise Exception(message)

        blackduck_opts.append(f"--detect.binary.scan.file.path={target_binary_path}")
        if not has_scan_target:
            blackduck_opts.append("--detect.blackduck.signature.scanner.disabled=true")

        return True, created_tars, blackduck_opts

    return has_scan_target, created_tars, blackduck_opts


@step_filter
def initiate_scan():
    print_script_start(__file__)
    hooks.run_hooks(__file__, "init")

    if os.environ.get("ENABLE_SECURITY_SCAN", "0") != "1":
        logger.info(
            "Security scan was not enabled, try setting ENABLE_SECURITY_SCAN=1 if wanted"
        )
        return

    workspace_adapter = get_workspace_adapter()

    logger.info("Security scan enabled, running")
    logger.info("Setting up BRANCH_TAG, VERSION, and PROJECT")

    scan_branch_tag = _get_scan_branch_tag(workspace_adapter)
    version = f"{scan_branch_tag}:{workspace_adapter.build_identifier}"
    project = f"{workspace_adapter.org}/{workspace_adapter.repo}"

    logger.info(
        f"Setup scan branch tag as {scan_branch_tag} and version: {version}, project: {project}"
    )

    blackduck_detector_search_depth = int(
        os.environ.get("BLACKDUCK_DETECTOR_SEARCH_DEPTH", "3")
    )

    has_scan_target, created_tars, blackduck_opts = _process_image(
        False, workspace_adapter
    )

    has_scan_target, added_tars, new_opts = _process_docker_images_list(
        has_scan_target, workspace_adapter
    )

    created_tars.extend(added_tars)
    blackduck_opts.extend(new_opts)

    has_scan_target, added_tars, new_blackduck_opts = _process_source_path(
        has_scan_target,
        blackduck_detector_search_depth,
        workspace_adapter.project_root_folder,
    )
    created_tars.extend(added_tars)
    blackduck_opts.extend(new_blackduck_opts)

    has_scan_target, added_tars, new_blackduck_opts = _process_binary_path(
        has_scan_target
    )
    created_tars.extend(added_tars)
    blackduck_opts.extend(new_blackduck_opts)

    if not has_scan_target:
        message = "Please specify either BLACKDUCK_TARGET_DOCKER_IMAGE, BLACKDUCK_TARGET_BINARY_PATH, or BLACKDUCK_TARGET_SOURCE_PATH environment variable"
        logger.warning(message)
        raise Exception(message)
    else:
        detect_result = requests.get("https://detect.synopsys.com/detect.sh")
        detect_result.raise_for_status()
        detect_path = Path(f"{tempfile.gettempdir()}/detect.sh")
        with open(detect_path, "wb") as detect_file:
            detect_file.write(detect_result.content)

        detect_path.chmod(0o755)
        blackduck_command = [str(detect_path)]
        blackduck_command.extend(blackduck_opts)
        run_command_with_output(blackduck_command)

        for file in created_tars:
            logger.info(f"Removing {file}")
            file.unlink(missing_ok=True)

        detect_path.unlink(missing_ok=True)

    print_script_end(__file__)


if __name__ == "__main__":
    initiate_scan()
