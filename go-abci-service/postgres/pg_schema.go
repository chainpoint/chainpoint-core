package postgres

import "github.com/chainpoint/chainpoint-core/go-abci-service/util"

var chainpointSchema = `--
-- PostgreSQL database dump
--

-- Dumped from database version 11.2 (Debian 11.2-1.pgdg90+1)
-- Dumped by pg_dump version 11.2 (Debian 11.2-1.pgdg90+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_with_oids = false;
`

var aggStatesSchema = `
--
-- Name: agg_states; Type: TABLE; Schema: public; Owner: chainpoint
--

CREATE TABLE public.agg_states (
    proof_id uuid NOT NULL,
    hash character varying(255),
    agg_id uuid,
    agg_state text,
    agg_root character varying(255),
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


ALTER TABLE public.agg_states OWNER TO chainpoint;
`

var btcAggStatesSchema = `
--
-- Name: anchor_btc_agg_states; Type: TABLE; Schema: public; Owner: chainpoint
--

CREATE TABLE public.anchor_btc_agg_states (
    cal_id character varying(255) NOT NULL,
    anchor_btc_agg_id uuid,
    anchor_btc_agg_state text,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


ALTER TABLE public.anchor_btc_agg_states OWNER TO chainpoint;
`

var btcHeadStatesSchema = `
--
-- Name: btchead_states; Type: TABLE; Schema: public; Owner: chainpoint
--

CREATE TABLE public.btchead_states (
    btctx_id character varying(255) NOT NULL,
    btchead_height integer,
    btchead_state text,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


ALTER TABLE public.btchead_states OWNER TO chainpoint;
`

var btcTxStatesSchema = `
--
-- Name: btctx_states; Type: TABLE; Schema: public; Owner: chainpoint
--

CREATE TABLE public.btctx_states (
    anchor_btc_agg_id uuid NOT NULL,
    btctx_id character varying(255),
    btctx_state text,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


ALTER TABLE public.btctx_states OWNER TO chainpoint;
`

var calStatesSchema = `
--
-- Name: cal_states; Type: TABLE; Schema: public; Owner: chainpoint
--

CREATE TABLE public.cal_states (
    agg_id uuid NOT NULL,
    cal_id character varying(255),
    cal_state text,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


ALTER TABLE public.cal_states OWNER TO chainpoint;
`

var proofsSchema = `
--
-- Name: proofs; Type: TABLE; Schema: public; Owner: chainpoint
--

CREATE TABLE public.proofs (
    proof_id uuid NOT NULL,
    proof text,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone NOT NULL
);


ALTER TABLE public.proofs OWNER TO chainpoint;
`

var primaryKeys = `
--
-- Name: agg_states agg_states_pkey; Type: CONSTRAINT; Schema: public; Owner: chainpoint
--

ALTER TABLE ONLY public.agg_states
    ADD CONSTRAINT agg_states_pkey PRIMARY KEY (proof_id);


--
-- Name: anchor_btc_agg_states anchor_btc_agg_states_pkey; Type: CONSTRAINT; Schema: public; Owner: chainpoint
--

ALTER TABLE ONLY public.anchor_btc_agg_states
    ADD CONSTRAINT anchor_btc_agg_states_pkey PRIMARY KEY (cal_id);


--
-- Name: btchead_states btchead_states_pkey; Type: CONSTRAINT; Schema: public; Owner: chainpoint
--

ALTER TABLE ONLY public.btchead_states
    ADD CONSTRAINT btchead_states_pkey PRIMARY KEY (btctx_id);


--
-- Name: btctx_states btctx_states_pkey; Type: CONSTRAINT; Schema: public; Owner: chainpoint
--

ALTER TABLE ONLY public.btctx_states
    ADD CONSTRAINT btctx_states_pkey PRIMARY KEY (anchor_btc_agg_id);


--
-- Name: cal_states cal_states_pkey; Type: CONSTRAINT; Schema: public; Owner: chainpoint
--

ALTER TABLE ONLY public.cal_states
    ADD CONSTRAINT cal_states_pkey PRIMARY KEY (agg_id);


--
-- Name: proofs proofs_pkey; Type: CONSTRAINT; Schema: public; Owner: chainpoint
--

ALTER TABLE ONLY public.proofs
    ADD CONSTRAINT proofs_pkey PRIMARY KEY (proof_id);
`

var createIndices = `

--
-- Name: agg_states_agg_id; Type: INDEX; Schema: public; Owner: chainpoint
--

CREATE INDEX agg_states_agg_id ON public.agg_states USING btree (agg_id);


--
-- Name: agg_states_created_at_agg_id_agg_root; Type: INDEX; Schema: public; Owner: chainpoint
--

CREATE INDEX agg_states_created_at_agg_id_agg_root ON public.agg_states USING btree (created_at, agg_id, agg_root);


--
-- Name: anchor_btc_agg_states_anchor_btc_agg_id; Type: INDEX; Schema: public; Owner: chainpoint
--

CREATE INDEX anchor_btc_agg_states_anchor_btc_agg_id ON public.anchor_btc_agg_states USING btree (anchor_btc_agg_id);


--
-- Name: anchor_btc_agg_states_created_at; Type: INDEX; Schema: public; Owner: chainpoint
--

CREATE INDEX anchor_btc_agg_states_created_at ON public.anchor_btc_agg_states USING btree (created_at);


--
-- Name: btchead_states_btchead_height; Type: INDEX; Schema: public; Owner: chainpoint
--

CREATE INDEX btchead_states_btchead_height ON public.btchead_states USING btree (btchead_height);


--
-- Name: btchead_states_created_at; Type: INDEX; Schema: public; Owner: chainpoint
--

CREATE INDEX btchead_states_created_at ON public.btchead_states USING btree (created_at);


--
-- Name: btctx_states_btctx_id; Type: INDEX; Schema: public; Owner: chainpoint
--

CREATE INDEX btctx_states_btctx_id ON public.btctx_states USING btree (btctx_id);


--
-- Name: btctx_states_created_at; Type: INDEX; Schema: public; Owner: chainpoint
--

CREATE INDEX btctx_states_created_at ON public.btctx_states USING btree (created_at);


--
-- Name: cal_states_cal_id; Type: INDEX; Schema: public; Owner: chainpoint
--

CREATE INDEX cal_states_cal_id ON public.cal_states USING btree (cal_id);


--
-- Name: cal_states_created_at; Type: INDEX; Schema: public; Owner: chainpoint
--

CREATE INDEX cal_states_created_at ON public.cal_states USING btree (created_at);


--
-- Name: proofs_created_at; Type: INDEX; Schema: public; Owner: chainpoint
--

CREATE INDEX proofs_created_at ON public.proofs USING btree (created_at);


--
-- PostgreSQL database dump complete
--
`

func (pg *Postgres) SchemaExists () bool {
	query := `SELECT to_regclass('proofs');`
	var res string
	rows, err := pg.DB.Query(query, &res)
	if util.LoggerError(pg.Logger, err) != nil {
		return false
	}
	if rows.Next() {
		pg.Logger.Info("Schema exists")
		return true
	}
	return false
}

func (pg *Postgres) CreateSchema () {
	tables := []string{chainpointSchema, aggStatesSchema, aggStatesSchema, btcAggStatesSchema, btcHeadStatesSchema, btcTxStatesSchema, calStatesSchema, proofsSchema, primaryKeys, createIndices}
	for _, tabl := range tables {
		go func(table string) {
			_, err := pg.DB.Exec(table)
			util.LoggerError(pg.Logger, err)
		}(tabl)
	}
}
