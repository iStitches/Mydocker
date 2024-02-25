package nsenter

/*
#define _GNU_SOURCE
#include <fcntl.h>
#include <errno.h>
#include <sched.h>
#include <unistd.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <signal.h>

__attribute__((constructor)) void enter_namespace(void) {
    fprintf(stdout, "Exec Cgo enter_namespace function\n");
    char *mydocker_pid;
    mydocker_pid = getenv("mydocker_pid");
    if (mydocker_pid) {
        fprintf(stdout, "got mydocker_pid=%s\n", mydocker_pid);
    } else {
        fprintf(stdout, "missing mydocker_pid env skip nsenter");
        return;
    }
    char *mydocker_cmd;
    // get execCommands from environment
    mydocker_cmd = getenv("mydocker_cmd");
    if (mydocker_cmd) {
        fprintf(stdout, "got mydocker_cmd=%s\n", mydocker_cmd);
    } else {
        fprintf(stdout, "missing mydocker_cmd env skip nsenter");
        return;
    }
    int i;
    char nspath[1024];
    char *namespaces[] = { "ipc", "uts", "net", "pid", "mnt" };
    for (i = 0; i < 5; i++) {
        sprintf(nspath, "/proc/%s/ns/%s", mydocker_pid, namespaces[i]);
        int fd = open(nspath, O_RDONLY);
        if (setns(fd, 0) == -1) {
            fprintf(stderr, "setns on %s namespace failed: %s\n", namespaces[i], strerror(errno));
        }
        close(fd);
    }
    int res = system(mydocker_cmd);
    exit(0);
    return;
}
*/
import "C"
