# Manual de uso

> Agrega capturas de pantalla en `docs/img/` cuando la interfaz esté terminada (tarea T13).

## Roles

| Rol | Puede |
|---|---|
| Visitante | Ver la cartelera, las funciones y los asientos disponibles |
| Usuario | Todo lo anterior, además de reservar, modificar y cancelar sus reservas |
| Administrador | Todo lo anterior, además de crear, editar y eliminar películas y funciones |

## Crear una cuenta e iniciar sesión

1. Abre <http://localhost:3000> y entra en **Registrarse**.
2. Escribe tu nombre, email y una contraseña de al menos 8 caracteres.
3. La sesión se inicia automáticamente. Por seguridad, dura **1 hora**; después tendrás que volver a iniciar sesión.

## Reservar asientos

1. En la **Cartelera**, elige una película.
2. Selecciona una **función** (fecha, hora y sala).
3. En el mapa de la sala:
   - Asiento libre: se puede seleccionar.
   - Asiento ocupado: está deshabilitado.
   - Asiento seleccionado: está resaltado.
4. Elige de 1 a 10 asientos y revisa el total.
5. Pulsa **Confirmar reserva**.

Si otra persona reservó uno de esos asientos un instante antes, verás el aviso *"Uno o más asientos ya fueron reservados"* y el mapa se actualizará para que elijas otros.

## Ver, modificar y cancelar

En **Mis reservas** aparecen todas tus reservas, activas y canceladas.

- **Modificar:** cambia los asientos de una reserva activa. Los nuevos asientos reemplazan a los anteriores.
- **Cancelar:** libera los asientos al momento. La reserva queda en tu historial como *Cancelada*.

Solo se puede modificar o cancelar **antes de que empiece la función**.

## Administración

Con una cuenta de administrador aparece el menú **Admin**:

- **Películas:** crear, editar o eliminar. No se puede eliminar una película con funciones que tengan reservas activas.
- **Funciones:** elegir película, sala, fecha y hora, y precio. No se puede eliminar una función con reservas activas.

## Cerrar sesión

Pulsa **Cerrar sesión** en el menú. El token se borra de este navegador.
