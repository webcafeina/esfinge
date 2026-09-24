-- Las invitaciones a quien todavía no tiene cuenta (ADR 0043, entrega B3).
--
-- Aquí **no hay ningún sobre**: el envío espera cifrado dentro de la bóveda de
-- quien lo manda y solo sale cuando esa persona crea su cuenta y publica sus
-- llaves. Lo único que se guarda aquí es que ya se le mandó un correo, para no
-- mandarle otro cada vez que el equipo que invitó vuelva a intentarlo.
--
-- Por eso la clave es (correo, quién invita): que dos personas distintas inviten
-- a la misma dirección son dos correos, y son legítimos los dos.
CREATE TABLE invitaciones (
	correo TEXT NOT NULL,
	de_cuenta TEXT NOT NULL,
	caduca INTEGER NOT NULL,
	PRIMARY KEY (correo, de_cuenta)
);

-- Para que la limpieza de cada hora no recorra la tabla entera.
CREATE INDEX invitaciones_caduca ON invitaciones (caduca);
