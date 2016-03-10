create table raw_question (
  id int unique not null,
  data jsonb not null
);

create table raw_answer (
  id int unique not null,
  q_id int not null references raw_question(id),
  user_id int not null,
  data jsonb not null
);
