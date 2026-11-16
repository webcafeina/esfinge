// Los procesos auxiliares de WebKit, los de dentro de la AppImage.
//
// WebKitGTK no vive en un solo proceso: la ventana habla con un `WebKitWebProcess` y un
// `WebKitNetworkProcess` que arranca ella misma. Y **la ruta de esos programas va compilada a
// fuego dentro de libwebkit** —en Debian y Ubuntu, /usr/lib/<triplete>/webkit2gtk-4.1/—, así que
// una AppImage que lleve WebKit dentro arranca los **del sistema de quien la ejecuta**.
//
// Eso da dos fallos que no se parecen y son el mismo:
//
//   · en un equipo **sin** WebKit instalado, el `execve` da ENOENT y la ventana no llega a
//     pintarse: «Unable to spawn a new child process»;
//   · en uno que **sí** lo tenga pero de otra versión —el caso normal, porque nuestro propio
//     `.deb` lo instala—, arrancan unos auxiliares que no hablan el mismo protocolo que nuestra
//     libwebkit, y eso sale como «Received invalid message: 'WebPage_SetActivityStateReply'» y un
//     SIGABRT. Lo vio el cliente en una Ubuntu 26.10 el 2026-11-16.
//
// En un equipo cuya versión coincida con la del empaquetado no se ve ninguno de los dos, que es
// por lo que esto sobrevivió a la publicación: la AppImage se arma en Ubuntu 24.04.
//
// **Y la variable que parecía resolverlo no existe.** `WEBKIT_EXEC_PATH` solo la mira WebKit en
// compilaciones de desarrollo; en la que empaquetan las distribuciones ni siquiera está la cadena
// dentro de la biblioteca —comprobado con `strings` sobre la libwebkit que llevamos dentro, donde
// sí aparecen `WEBKIT_INJECTED_BUNDLE_PATH` y las `WEBKIT_SKIA_*` y ésa no—. Exportarla no hacía
// nada, y el comentario que decía que sí era parte del fallo.
//
// Así que lo que queda es interponerse: este objeto se carga con LD_PRELOAD y reescribe la ruta
// del programa **justo en la llamada que lo lanza**, hacia la copia que la AppImage lleva dentro.
//
// Es a propósito lo más estrecho que se puede: sólo toca una ruta absoluta, que nombre un
// directorio `webkit2gtk-`, y **sólo si dentro del AppDir existe ese mismo fichero y es
// ejecutable**. Cualquier otra cosa se pasa tal cual al original. Fuera de una AppImage no hay
// APPDIR y entonces no hace absolutamente nada, que es el caso del `.deb` y del `.rpm`.
//
// Se interponen los cinco de la familia `exec*` que pueden traer la ruta ya resuelta, más
// `posix_spawn`: glib ha usado unos u otros según la versión, y acertar con cuál toca hoy sería
// justo la clase de suposición que ya costó este fallo.

#define _GNU_SOURCE
#include <dlfcn.h>
#include <spawn.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/types.h>
#include <unistd.h>

// La ruta de dentro, si la hay. Devuelve `ruta` tal cual cuando no hay nada que reescribir, así
// que quien llama puede usar el resultado sin preguntar.
static const char *de_dentro(const char *ruta, char *buf, size_t n) {
    const char *appdir = getenv("APPDIR");
    if (appdir == NULL || *appdir == '\0') return ruta;   // no estamos en una AppImage
    if (ruta == NULL || ruta[0] != '/') return ruta;      // relativa: no es la que nos ocupa
    if (strstr(ruta, "/webkit2gtk-") == NULL) return ruta;
    if (snprintf(buf, n, "%s%s", appdir, ruta) >= (int)n) return ruta;
    if (access(buf, X_OK) != 0) return ruta;              // dentro no está: que siga su camino
    return buf;
}

// Cada interposición hace lo mismo: pedirle al enlazador la de verdad —la que está detrás de la
// nuestra— y llamarla con la ruta ya mirada. Van con su `typedef` y sin macros: una firma de
// puntero a función dentro de un macro se lee tres veces peor que escribirla.

typedef int (*exec_t)(const char *, char *const[]);
typedef int (*exece_t)(const char *, char *const[], char *const[]);
typedef int (*spawn_t)(pid_t *, const char *, const posix_spawn_file_actions_t *,
                       const posix_spawnattr_t *, char *const[], char *const[]);

int execv(const char *ruta, char *const argv[]) {
    static exec_t real;
    if (real == NULL) real = (exec_t)dlsym(RTLD_NEXT, "execv");
    if (real == NULL) return -1;
    char buf[4096];
    return real(de_dentro(ruta, buf, sizeof buf), argv);
}

int execvp(const char *ruta, char *const argv[]) {
    static exec_t real;
    if (real == NULL) real = (exec_t)dlsym(RTLD_NEXT, "execvp");
    if (real == NULL) return -1;
    char buf[4096];
    return real(de_dentro(ruta, buf, sizeof buf), argv);
}

int execve(const char *ruta, char *const argv[], char *const envp[]) {
    static exece_t real;
    if (real == NULL) real = (exece_t)dlsym(RTLD_NEXT, "execve");
    if (real == NULL) return -1;
    char buf[4096];
    return real(de_dentro(ruta, buf, sizeof buf), argv, envp);
}

int execvpe(const char *ruta, char *const argv[], char *const envp[]) {
    static exece_t real;
    if (real == NULL) real = (exece_t)dlsym(RTLD_NEXT, "execvpe");
    if (real == NULL) return -1;
    char buf[4096];
    return real(de_dentro(ruta, buf, sizeof buf), argv, envp);
}

int posix_spawn(pid_t *pid, const char *ruta, const posix_spawn_file_actions_t *acciones,
                const posix_spawnattr_t *atributos, char *const argv[], char *const envp[]) {
    static spawn_t real;
    if (real == NULL) real = (spawn_t)dlsym(RTLD_NEXT, "posix_spawn");
    if (real == NULL) return -1;
    char buf[4096];
    return real(pid, de_dentro(ruta, buf, sizeof buf), acciones, atributos, argv, envp);
}

int posix_spawnp(pid_t *pid, const char *ruta, const posix_spawn_file_actions_t *acciones,
                 const posix_spawnattr_t *atributos, char *const argv[], char *const envp[]) {
    static spawn_t real;
    if (real == NULL) real = (spawn_t)dlsym(RTLD_NEXT, "posix_spawnp");
    if (real == NULL) return -1;
    char buf[4096];
    return real(pid, de_dentro(ruta, buf, sizeof buf), acciones, atributos, argv, envp);
}
