DROP SCHEMA IF EXISTS auto_org_invitation;
CREATE SCHEMA auto_org_invitation;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";


DROP TYPE IF EXISTS invitation_status;
CREATE TYPE invitation_status AS ENUM ('PENDING', 'FAILED', 'SUCCEEDED');

CREATE TABLE auto_org_invitation.invitations (
    id uuid NOT NULL,
    order_id BIGINT NOT NULL,
    github_username CHARACTER VARYING COLLATE pg_catalog."default" NOT NULL,
    github_email CHARACTER VARYING COLLATE pg_catalog."default" NOT NULL,
    invitation_status invitation_status NOT NULL,
    first_error TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT current_timestamp,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT 'epoch'::timestamp,
    CONSTRAINT pk PRIMARY KEY (id)
);

CREATE TABLE auto_org_invitation.failed_invitations (
    id uuid NOT NULL,
    order_id BIGINT NOT NULL,
    github_username CHARACTER VARYING COLLATE pg_catalog."default" NOT NULL,
    github_email CHARACTER VARYING COLLATE pg_catalog."default" NOT NULL,
    invitation_status invitation_status NOT NULL,
    failed_at TIMESTAMP WITH TIME ZONE DEFAULT current_timestamp
);

CREATE TABLE auto_org_invitation.successful_invitations (
    id uuid NOT NULL,
    order_id BIGINT NOT NULL,
    github_username CHARACTER VARYING COLLATE pg_catalog."default" NOT NULL,
    github_email CHARACTER VARYING COLLATE pg_catalog."default" NOT NULL,
    invitation_status invitation_status NOT NULL,
    succeeded_at TIMESTAMP WITH TIME ZONE DEFAULT current_timestamp
);

CREATE OR REPLACE FUNCTION auto_org_invitation.check_status()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.invitation_status = 'SUCCEEDED' THEN
        INSERT INTO auto_org_invitation.successful_invitations (id, order_id, github_username, github_email, invitation_status)
        VALUES (NEW.id, NEW.order_id, NEW.github_username, NEW.github_email, NEW.invitation_status);
    END IF;
    IF NEW.invitation_status = 'FAILED' THEN
        INSERT INTO auto_org_invitation.failed_invitations (id, order_id, github_username, github_email, invitation_status)
        VALUES (NEW.id, NEW.order_id, NEW.github_username, NEW.github_email, NEW.invitation_status);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER  check_status_trigger
AFTER INSERT OR UPDATE ON auto_org_invitation.invitations
FOR EACH ROW EXECUTE FUNCTION auto_org_invitation.check_status();