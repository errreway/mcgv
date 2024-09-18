package ru.gitverse.sbertech.client.filters;

import org.apache.ignite.lang.IgniteBiPredicate;

public class PersonByNamePojoFilter implements IgniteBiPredicate<Long, Person> {
    private String name;

    public PersonByNamePojoFilter() {

    }

    public PersonByNamePojoFilter(String name) {
        this.name = name;
    }

    @Override
    public boolean apply(Long id, Person person) {
        return person != null && person.getName() != null && name.equals(person.getName());
    }
}