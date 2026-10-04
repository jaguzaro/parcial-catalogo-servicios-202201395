-- +goose Up
CREATE TABLE importacion (
    id             bigint GENERATED ALWAYS AS IDENTITY,
    usuario_id     bigint,
    origen         text        NOT NULL,
    estado         text        NOT NULL,
    archivo_ruta   text        NOT NULL,
    archivo_sha256 text        NOT NULL,
    hoja           text        NOT NULL,
    creados        integer     NOT NULL DEFAULT 0,
    actualizados   integer     NOT NULL DEFAULT 0,
    omitidos       integer     NOT NULL DEFAULT 0,
    observados     integer     NOT NULL DEFAULT 0,
    detalle        jsonb       NOT NULL DEFAULT '{}',
    mensaje_error  text,
    iniciada_en    timestamptz NOT NULL DEFAULT now(),
    finalizada_en  timestamptz,
    CONSTRAINT importacion_pk PRIMARY KEY (id),
    CONSTRAINT importacion_usuario_fk FOREIGN KEY (usuario_id) REFERENCES usuario (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT importacion_origen_ck CHECK (origen IN ('cli', 'web')),
    CONSTRAINT importacion_origen_usuario_ck CHECK ((origen = 'web') = (usuario_id IS NOT NULL)),
    CONSTRAINT importacion_estado_ck CHECK (estado IN ('en_curso', 'completada', 'fallida')),
    CONSTRAINT importacion_sha256_ck CHECK (archivo_sha256 ~ '^[0-9a-f]{64}$'),
    CONSTRAINT importacion_creados_ck CHECK (creados >= 0),
    CONSTRAINT importacion_actualizados_ck CHECK (actualizados >= 0),
    CONSTRAINT importacion_omitidos_ck CHECK (omitidos >= 0),
    CONSTRAINT importacion_observados_ck CHECK (observados >= 0),
    CONSTRAINT importacion_error_ck CHECK ((estado = 'fallida') = (mensaje_error IS NOT NULL)),
    CONSTRAINT importacion_fin_ck CHECK ((estado = 'en_curso') = (finalizada_en IS NULL))
);
CREATE INDEX importacion_usuario_idx ON importacion (usuario_id);

-- +goose Down
DROP TABLE importacion;
