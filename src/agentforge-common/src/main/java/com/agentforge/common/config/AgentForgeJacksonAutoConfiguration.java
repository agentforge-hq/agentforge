package com.agentforge.common.config;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.springframework.boot.autoconfigure.AutoConfiguration;
import org.springframework.boot.autoconfigure.condition.ConditionalOnClass;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Primary;

import java.util.TimeZone;

/**
 * Auto-configuration for Jackson ObjectMapper adhering to platform REST
 * contracts:
 * - Omit null fields (NON_NULL)
 * - ISO-8601 UTC dates (disable timestamp numbers)
 * - Register JavaTimeModule for java.time types
 */
@AutoConfiguration
@ConditionalOnClass(ObjectMapper.class)
public class AgentForgeJacksonAutoConfiguration {

    @Bean
    @Primary
    @ConditionalOnMissingBean
    public ObjectMapper objectMapper() {
        ObjectMapper mapper = new ObjectMapper();
        mapper.registerModule(new JavaTimeModule());
        mapper.disable(SerializationFeature.WRITE_DATES_AS_TIMESTAMPS);
        mapper.setDefaultPropertyInclusion(
                JsonInclude.Value.construct(JsonInclude.Include.NON_NULL, JsonInclude.Include.ALWAYS));
        mapper.setTimeZone(TimeZone.getTimeZone("UTC"));
        return mapper;
    }
}
