create table raw_question (
  id int unique not null,
  data jsonb not null
);

create table raw_answer (
  id int unique not null,
  user_id int not null,
  data jsonb not null
);
