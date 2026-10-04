-- +goose Up
CREATE TABLE clase_servicio (
    id             smallint GENERATED ALWAYS AS IDENTITY,
    nombre         text        NOT NULL,
    valor_origen   text,
    origen_celda   text,
    orden          smallint    NOT NULL,
    activo         boolean     NOT NULL DEFAULT true,
    importacion_id bigint,
    creado_en      timestamptz NOT NULL DEFAULT now(),
    actualizado_en timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT clase_servicio_pk PRIMARY KEY (id),
    CONSTRAINT clase_servicio_nombre_uk UNIQUE (nombre),
    CONSTRAINT clase_servicio_valor_origen_uk UNIQUE (valor_origen),
    CONSTRAINT clase_servicio_nombre_ck CHECK (btrim(nombre) <> ''),
    CONSTRAINT clase_servicio_origen_ck CHECK ((valor_origen IS NULL) = (origen_celda IS NULL)),
    CONSTRAINT clase_servicio_orden_ck CHECK (orden > 0),
    CONSTRAINT clase_servicio_importacion_fk FOREIGN KEY (importacion_id) REFERENCES importacion (id) ON DELETE RESTRICT ON UPDATE RESTRICT
);
CREATE INDEX clase_servicio_importacion_idx ON clase_servicio (importacion_id);

CREATE TABLE criticidad (
    id             smallint GENERATED ALWAYS AS IDENTITY,
    nombre         text        NOT NULL,
    valor_origen   text,
    origen_celda   text,
    orden          smallint    NOT NULL,
    activo         boolean     NOT NULL DEFAULT true,
    importacion_id bigint,
    creado_en      timestamptz NOT NULL DEFAULT now(),
    actualizado_en timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT criticidad_pk PRIMARY KEY (id),
    CONSTRAINT criticidad_nombre_uk UNIQUE (nombre),
    CONSTRAINT criticidad_valor_origen_uk UNIQUE (valor_origen),
    CONSTRAINT criticidad_nombre_ck CHECK (btrim(nombre) <> ''),
    CONSTRAINT criticidad_origen_ck CHECK ((valor_origen IS NULL) = (origen_celda IS NULL)),
    CONSTRAINT criticidad_orden_ck CHECK (orden > 0),
    CONSTRAINT criticidad_importacion_fk FOREIGN KEY (importacion_id) REFERENCES importacion (id) ON DELETE RESTRICT ON UPDATE RESTRICT
);
CREATE INDEX criticidad_importacion_idx ON criticidad (importacion_id);

CREATE TABLE tipo_servicio (
    id             smallint GENERATED ALWAYS AS IDENTITY,
    nombre         text        NOT NULL,
    valor_origen   text,
    origen_celda   text,
    orden          smallint    NOT NULL,
    activo         boolean     NOT NULL DEFAULT true,
    importacion_id bigint,
    creado_en      timestamptz NOT NULL DEFAULT now(),
    actualizado_en timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT tipo_servicio_pk PRIMARY KEY (id),
    CONSTRAINT tipo_servicio_nombre_uk UNIQUE (nombre),
    CONSTRAINT tipo_servicio_valor_origen_uk UNIQUE (valor_origen),
    CONSTRAINT tipo_servicio_nombre_ck CHECK (btrim(nombre) <> ''),
    CONSTRAINT tipo_servicio_origen_ck CHECK ((valor_origen IS NULL) = (origen_celda IS NULL)),
    CONSTRAINT tipo_servicio_orden_ck CHECK (orden > 0),
    CONSTRAINT tipo_servicio_importacion_fk FOREIGN KEY (importacion_id) REFERENCES importacion (id) ON DELETE RESTRICT ON UPDATE RESTRICT
);
CREATE INDEX tipo_servicio_importacion_idx ON tipo_servicio (importacion_id);

CREATE TABLE servicio_n1 (
    id             bigint GENERATED ALWAYS AS IDENTITY,
    codigo         text        NOT NULL,
    nombre         text        NOT NULL,
    activo         boolean     NOT NULL DEFAULT true,
    origen_hoja    text,
    origen_rango   text,
    origen_valores jsonb,
    origen_hash    text,
    importacion_id bigint,
    creado_en      timestamptz NOT NULL DEFAULT now(),
    actualizado_en timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT servicio_n1_pk PRIMARY KEY (id),
    CONSTRAINT servicio_n1_codigo_uk UNIQUE (codigo),
    CONSTRAINT servicio_n1_codigo_ck CHECK (btrim(codigo) <> ''),
    CONSTRAINT servicio_n1_nombre_ck CHECK (btrim(nombre) <> ''),
    CONSTRAINT servicio_n1_hash_ck CHECK (origen_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT servicio_n1_origen_ck CHECK (
        (origen_hoja IS NULL) = (origen_hash IS NULL) AND (origen_hoja IS NULL) = (origen_valores IS NULL)),
    CONSTRAINT servicio_n1_importacion_fk FOREIGN KEY (importacion_id) REFERENCES importacion (id) ON DELETE RESTRICT ON UPDATE RESTRICT
);
CREATE INDEX servicio_n1_nombre_idx ON servicio_n1 (nombre);
CREATE INDEX servicio_n1_activo_idx ON servicio_n1 (activo);
CREATE INDEX servicio_n1_importacion_idx ON servicio_n1 (importacion_id);

CREATE TABLE servicio_n2 (
    id                     bigint GENERATED ALWAYS AS IDENTITY,
    servicio_n1_id         bigint      NOT NULL,
    codigo                 text        NOT NULL,
    nombre                 text        NOT NULL,
    activo                 text        NOT NULL DEFAULT 'S',
    clase_id               smallint,
    criticidad_id          smallint,
    tipo_id                smallint,
    descripcion            text,
    metrica                text,
    minimo                 numeric,
    maximo                 numeric,
    seccion_responsable_id bigint,
    usuario_responsable_id bigint,
    requiere_revision      boolean     NOT NULL DEFAULT false,
    origen_hoja            text,
    origen_rango           text,
    origen_valores         jsonb,
    origen_hash            text,
    importacion_id         bigint,
    creado_en              timestamptz NOT NULL DEFAULT now(),
    actualizado_en         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT servicio_n2_pk PRIMARY KEY (id),
    CONSTRAINT servicio_n2_n1_fk FOREIGN KEY (servicio_n1_id) REFERENCES servicio_n1 (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT servicio_n2_codigo_uk UNIQUE (codigo),
    CONSTRAINT servicio_n2_codigo_ck CHECK (btrim(codigo) <> ''),
    CONSTRAINT servicio_n2_nombre_ck CHECK (btrim(nombre) <> ''),
    CONSTRAINT servicio_n2_activo_ck CHECK (activo IN ('S', 'N', 'DESCONOCIDO')),
    CONSTRAINT servicio_n2_clase_fk FOREIGN KEY (clase_id) REFERENCES clase_servicio (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT servicio_n2_criticidad_fk FOREIGN KEY (criticidad_id) REFERENCES criticidad (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT servicio_n2_tipo_fk FOREIGN KEY (tipo_id) REFERENCES tipo_servicio (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT servicio_n2_minmax_ck CHECK (minimo IS NULL OR maximo IS NULL OR minimo <= maximo),
    CONSTRAINT servicio_n2_seccion_fk FOREIGN KEY (seccion_responsable_id) REFERENCES seccion (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT servicio_n2_usuario_fk FOREIGN KEY (usuario_responsable_id) REFERENCES usuario (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT servicio_n2_responsable_ck CHECK (usuario_responsable_id IS NULL OR seccion_responsable_id IS NOT NULL),
    CONSTRAINT servicio_n2_hash_ck CHECK (origen_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT servicio_n2_origen_ck CHECK (
        (origen_hoja IS NULL) = (origen_hash IS NULL) AND (origen_hoja IS NULL) = (origen_valores IS NULL)),
    CONSTRAINT servicio_n2_importacion_fk FOREIGN KEY (importacion_id) REFERENCES importacion (id) ON DELETE RESTRICT ON UPDATE RESTRICT
);
CREATE INDEX servicio_n2_n1_idx ON servicio_n2 (servicio_n1_id);
CREATE INDEX servicio_n2_nombre_idx ON servicio_n2 (nombre);
CREATE INDEX servicio_n2_activo_idx ON servicio_n2 (activo);
CREATE INDEX servicio_n2_clase_idx ON servicio_n2 (clase_id);
CREATE INDEX servicio_n2_criticidad_idx ON servicio_n2 (criticidad_id);
CREATE INDEX servicio_n2_tipo_idx ON servicio_n2 (tipo_id);
CREATE INDEX servicio_n2_seccion_idx ON servicio_n2 (seccion_responsable_id);
CREATE INDEX servicio_n2_usuario_idx ON servicio_n2 (usuario_responsable_id);
CREATE INDEX servicio_n2_importacion_idx ON servicio_n2 (importacion_id);

CREATE TABLE incidencia (
    id             bigint GENERATED ALWAYS AS IDENTITY,
    importacion_id bigint      NOT NULL,
    tipo           text        NOT NULL,
    regla          text,
    hoja           text        NOT NULL,
    fila           integer,
    celdas         text,
    codigo         text,
    servicio_n1_id bigint,
    servicio_n2_id bigint,
    valor_original jsonb,
    valor_aplicado jsonb,
    mensaje        text        NOT NULL,
    creada_en      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT incidencia_pk PRIMARY KEY (id),
    CONSTRAINT incidencia_importacion_fk FOREIGN KEY (importacion_id) REFERENCES importacion (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT incidencia_tipo_ck CHECK (tipo IN (
        'N1_HEREDADO_DE_FILA_ANTERIOR', 'N1_NOMBRE_CONFLICTO', 'CODIGO_FORMATO_NO_ESTANDAR',
        'ATRIBUTOS_AUSENTES', 'FILA_SIN_CODIGO', 'VALOR_FUERA_DE_DOMINIO', 'VALOR_FUERA_DE_CATALOGO',
        'VALOR_NUMERICO_INVALIDO', 'MINIMO_MAYOR_QUE_MAXIMO', 'CONFLICTO_ATRIBUTO',
        'PREFIJO_INCOHERENTE', 'CODIGO_N2_REPETIDO', 'PADRE_INACTIVO',
        'CODIGO_EXISTENTE_NO_IMPORTADO', 'EDICION_LOCAL_SOBRESCRITA', 'AUSENTE_EN_ARCHIVO')),
    CONSTRAINT incidencia_regla_ck CHECK (regla ~ '^D[0-9]{2}(\.[0-9]+)?$'),
    CONSTRAINT incidencia_fila_ck CHECK (fila > 0),
    CONSTRAINT incidencia_n1_fk FOREIGN KEY (servicio_n1_id) REFERENCES servicio_n1 (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT incidencia_n2_fk FOREIGN KEY (servicio_n2_id) REFERENCES servicio_n2 (id) ON DELETE RESTRICT ON UPDATE RESTRICT,
    CONSTRAINT incidencia_mensaje_ck CHECK (btrim(mensaje) <> '')
);
CREATE INDEX incidencia_importacion_idx ON incidencia (importacion_id);
CREATE INDEX incidencia_n1_idx ON incidencia (servicio_n1_id);
CREATE INDEX incidencia_n2_idx ON incidencia (servicio_n2_id);

-- +goose Down
DROP TABLE incidencia;
DROP TABLE servicio_n2;
DROP TABLE servicio_n1;
DROP TABLE tipo_servicio;
DROP TABLE criticidad;
DROP TABLE clase_servicio;
